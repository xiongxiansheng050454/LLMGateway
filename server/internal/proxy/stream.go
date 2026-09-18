package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/store"
)

var errDownstreamWrite = errors.New("downstream stream write failed")

type completionStream struct {
	service           *Service
	body              io.ReadCloser
	ctx               context.Context
	requestID         string
	auth              *domain.AuthContext
	candidate         domain.RouteCandidate
	publicModel       string
	clientIP          string
	start             time.Time
	reservationID     int64
	rateReservationID int64
	cancel            context.CancelFunc

	mu        sync.Mutex
	forwarded bool
}

func (s *completionStream) Close() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.rateReservationID != 0 {
		_ = s.service.store.ReleaseRateLimit(context.Background(), s.rateReservationID)
	}
	return s.body.Close()
}

func (s *completionStream) Forward(emit func([]byte) error) error {
	if s.cancel != nil {
		defer s.cancel()
	}
	s.mu.Lock()
	if s.forwarded {
		s.mu.Unlock()
		return errors.New("chat stream already forwarded")
	}
	s.forwarded = true
	s.mu.Unlock()
	defer s.body.Close()
	settled := false
	defer func() {
		if !settled && s.reservationID != 0 {
			_ = s.service.store.ReleaseQuota(context.Background(), s.reservationID)
		}
	}()

	var usage *Usage
	var ttft *int
	done := false
	err := s.service.adapter.ParseStream(s.body, s.publicModel, func(event StreamEvent) error {
		if event.Done {
			done = true
			return nil
		}
		if event.Data && ttft == nil {
			value := elapsedMs(s.start, s.service.now())
			ttft = &value
		}
		if event.Usage != nil {
			usage = event.Usage
		}
		if err := emit(event.Frame); err != nil {
			return fmt.Errorf("%w: %v", errDownstreamWrite, err)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errDownstreamWrite) || errors.Is(s.ctx.Err(), context.Canceled) {
			s.logError(usage, ttft, "client_canceled")
			return err
		}
		reason := domain.FailureUpstreamUnreachable
		code := "upstream_stream_interrupted"
		message := "upstream stream was interrupted"
		if errors.Is(err, ErrInvalidStream) {
			reason = domain.FailureUpstreamProtocol
			code = "upstream_stream_protocol_error"
			message = "upstream stream contained invalid data"
		}
		s.service.recordChannelHealth(s.candidate.ChannelID, false, reason)
		s.logError(usage, ttft, code)
		s.emitError(emit, code, message)
		return err
	}
	if !done {
		if errors.Is(s.ctx.Err(), context.Canceled) {
			s.logError(usage, ttft, "client_canceled")
			return s.ctx.Err()
		}
		s.service.recordChannelHealth(s.candidate.ChannelID, false, domain.FailureUpstreamProtocol)
		s.logError(usage, ttft, "upstream_stream_interrupted")
		s.emitError(emit, "upstream_stream_interrupted", "upstream stream ended before [DONE]")
		return ErrUpstream
	}
	if usage == nil {
		s.service.recordChannelHealth(s.candidate.ChannelID, false, domain.FailureUpstreamProtocol)
		s.logError(nil, ttft, "upstream_usage_missing")
		s.emitError(emit, "upstream_usage_missing", "upstream stream did not include usage")
		return ErrUpstream
	}

	durationMs := elapsedMs(s.start, s.service.now())
	cost, inputPrice, outputPrice, err := s.service.priceFor(s.candidate.ChannelID, s.publicModel, usage)
	if err != nil {
		s.logError(usage, ttft, "pricing_error")
		s.emitError(emit, "pricing_error", "unable to price completion")
		return err
	}
	s.service.recordChannelHealth(s.candidate.ChannelID, true, "")
	usageLog := s.service.usageLogInput(s.requestID, s.auth, &s.candidate.ChannelID, s.candidate.UpstreamModel, s.publicModel, usage, cost, inputPrice, outputPrice, durationMs, s.clientIP, "success", "")
	usageLog.TTFTMs = ttft
	_, err = s.service.store.SettleChatCompletion(domain.ChatSettlementInput{
		ReservationID: s.reservationID,
		UserID:        s.auth.UserID, APIKeyID: s.auth.KeyID, ChannelID: &s.candidate.ChannelID, Cost: cost,
		DebitChannel: s.candidate.Balance != nil, Description: "chat completion " + s.requestID, UsageLog: usageLog,
	})
	if err != nil {
		code := "settlement_failed"
		message := "unable to settle completion"
		if errors.Is(err, store.ErrInvalid) {
			code = "insufficient_balance"
			message = "insufficient balance"
		}
		s.logError(usage, ttft, code)
		s.emitError(emit, code, message)
		return err
	}
	settled = true
	if s.rateReservationID != 0 {
		_ = s.service.store.FinalizeRateLimit(context.Background(), s.rateReservationID, int64(usage.TotalTokens))
		s.rateReservationID = 0
	}
	_ = s.service.store.UpdateKeyLastUsed(s.auth.KeyID)
	if err := emit([]byte("data: [DONE]\n\n")); err != nil {
		return fmt.Errorf("%w: %v", errDownstreamWrite, err)
	}
	return nil
}

func (s *completionStream) logError(usage *Usage, ttft *int, code string) {
	durationMs := elapsedMs(s.start, s.service.now())
	input := s.service.usageLogInput(s.requestID, s.auth, &s.candidate.ChannelID, s.candidate.UpstreamModel, s.publicModel, usage, "0.000000", "", "", durationMs, s.clientIP, "error", code)
	input.TTFTMs = ttft
	_, _ = s.service.store.InsertUsageLog(input)
}

func (s *completionStream) emitError(emit func([]byte) error, code, message string) {
	_ = emit(s.service.adapter.StreamError(code, message))
	_ = emit([]byte("data: [DONE]\n\n"))
}
