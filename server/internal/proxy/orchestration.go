package proxy

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/money"
	"LLMGateway/server/internal/store"
)

// models returns the OpenAI-style model list visible to the key, filtered by
// its permissions.
func (a *Service) Models(auth *domain.AuthContext) (ModelList, error) {
	result, err := a.store.ListCatalogModels(true)
	if err != nil {
		return ModelList{}, err
	}

	created := a.now().Unix()
	seen := map[string]bool{}
	data := []Model{}
	for _, item := range result.List {
		name := item.ModelName
		if name == "" || seen[name] || !allowModel(auth, name) {
			continue
		}
		seen[name] = true
		data = append(data, Model{ID: name, Created: created, OwnedBy: "llmgateway"})
	}
	return ModelList{Models: data}, nil
}

// ChatCompletions prepares a buffered or streaming chat completion. A non-nil
// error is a pre-flight/transport failure the handler maps to an OpenAI error.
func (a *Service) ChatCompletions(ctx context.Context, auth *domain.AuthContext, req ChatRequest, clientIP string) (ChatResponse, error) {
	requestCtx, cancel := context.WithTimeout(ctx, a.requestTimeout)
	streamOwnsCancel := false
	defer func() {
		if !streamOwnsCancel {
			cancel()
		}
	}()
	ctx = requestCtx
	if req.Model == "" {
		return ChatResponse{}, ErrInvalidRequest
	}
	if !allowModel(auth, req.Model) {
		return ChatResponse{}, ErrForbidden
	}
	balance, err := money.Parse6(auth.AvailableBalance)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("%w: invalid balance", store.ErrInvalid)
	}
	if balance.Cmp(0) <= 0 {
		return ChatResponse{}, ErrInsufficientBalance
	}

	requestID := newRequestID()
	start := a.now()
	estimate, estimateErr := a.adapter.EstimateUsage(req.Body, a.defaultMaxTokens)
	var estimatedTokens *int64
	if estimateErr == nil {
		value := int64(estimate.TotalTokens)
		estimatedTokens = &value
	}
	if err := a.checkRateLimit(auth, req.Model, estimatedTokens); err != nil {
		if errors.Is(err, ErrRateLimited) {
			a.logUsage(requestID, auth, nil, "", req.Model, nil, "0.000000", "", "", elapsedMs(start, a.now()), clientIP, "error", "rate_limited")
		}
		return ChatResponse{}, err
	}
	if estimateErr != nil {
		return ChatResponse{}, ErrRateLimited
	}
	rateReservation, rateErr := a.store.ReserveRateLimit(ctx, domain.RateLimitReservationInput{RequestID: requestID, UserID: auth.UserID, APIKeyID: auth.KeyID, Model: req.Model, EstimatedTokens: int64(estimate.TotalTokens), ExpiresAt: a.now().Add(a.requestTimeout)})
	if rateErr != nil {
		return ChatResponse{}, ErrRateLimited
	}
	rateReservationOpen := true
	defer func() {
		if rateReservationOpen {
			_ = a.store.ReleaseRateLimit(context.Background(), rateReservation.ID)
		}
	}()

	candidates, err := a.orderedCandidates(req.Model)
	if err != nil {
		if errors.Is(err, ErrNoHealthyChannel) {
			// Degraded: every candidate is tripped open or there is no mapping.
			a.logUsage(requestID, auth, nil, "", req.Model, nil, "0.000000", "", "", elapsedMs(start, a.now()), clientIP, "error", "no_healthy_channel")
		}
		return ChatResponse{}, err
	}
	if len(candidates) == 0 {
		a.logUsage(requestID, auth, nil, "", req.Model, nil, "0.000000", "", "", elapsedMs(start, a.now()), clientIP, "error", "no_healthy_channel")
		return ChatResponse{}, ErrNoHealthyChannel
	}
	candidate := candidates[0]
	if a.maxAttempts < len(candidates) {
		candidates = candidates[:a.maxAttempts]
	}
	reservation, err := a.reserveQuota(ctx, requestID, auth, req, candidate.ChannelID)
	if err != nil {
		if errors.Is(err, ErrQuotaExceeded) {
			a.logUsage(requestID, auth, &candidate.ChannelID, candidate.UpstreamModel, req.Model, nil, "0.000000", "", "", elapsedMs(start, a.now()), clientIP, "error", "quota_exceeded")
		}
		return ChatResponse{}, err
	}
	releaseReservation := reservation.ID != 0
	defer func() {
		if releaseReservation {
			_ = a.store.ReleaseQuota(context.Background(), reservation.ID)
		}
	}()

	if a.adapter.RewriteRequest == nil {
		return ChatResponse{}, ErrInvalidRequest
	}
	var resp *http.Response
	var responseBody []byte
	var readErr error
	healthRecorded := false
	for attempt, next := range candidates {
		if ctx.Err() != nil {
			return ChatResponse{}, ctx.Err()
		}
		candidate = next
		if err := a.checkChannelRateLimit(auth, req.Model, candidate.ChannelID, *estimatedTokens); err != nil {
			if errors.Is(err, ErrRateLimited) {
				a.recordChannelHealth(candidate.ChannelID, false, domain.FailureUpstreamUnreachable)
				if attempt+1 < len(candidates) {
					continue
				}
				a.logUsage(requestID, auth, &candidate.ChannelID, candidate.UpstreamModel, req.Model, nil, "0.000000", "", "", elapsedMs(start, a.now()), clientIP, "error", "rate_limited")
			}
			return ChatResponse{}, err
		}
		secret, secretErr := a.store.GetChannelSecret(candidate.ChannelID)
		if secretErr != nil {
			return ChatResponse{}, secretErr
		}
		upstreamBody, rewriteErr := a.adapter.RewriteRequest(req.Body, candidate.UpstreamModel)
		if rewriteErr != nil {
			return ChatResponse{}, ErrInvalidRequest
		}
		httpReq, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(secret.BaseURL, "/")+"/v1/chat/completions", bytes.NewReader(upstreamBody))
		if requestErr != nil {
			return ChatResponse{}, ErrUpstream
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if req.Stream {
			httpReq.Header.Set("Accept", "text/event-stream")
		} else {
			httpReq.Header.Set("Accept", "application/json")
		}
		if secret.AuthType == "bearer" && secret.APIKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+secret.APIKey)
		}
		resp, err = a.client.Do(httpReq)
		if err != nil {
			if ctx.Err() != nil {
				return ChatResponse{}, ctx.Err()
			}
			reason, retry := classifyUpstreamResult(0, err)
			if retry {
				a.recordChannelHealth(candidate.ChannelID, false, reason)
			}
			if retry && attempt+1 < len(candidates) {
				continue
			}
			a.logUsage(requestID, auth, &candidate.ChannelID, candidate.UpstreamModel, req.Model, nil, "0.000000", "", "", elapsedMs(start, a.now()), clientIP, "error", "upstream_unreachable")
			return ChatResponse{}, ErrUpstream
		}
		if req.Stream && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			break
		}
		responseBody, readErr = io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			if ctx.Err() != nil {
				return ChatResponse{}, ctx.Err()
			}
			a.recordChannelHealth(candidate.ChannelID, false, domain.FailureUpstreamUnreachable)
			if attempt+1 < len(candidates) {
				continue
			}
			a.logUsage(requestID, auth, &candidate.ChannelID, candidate.UpstreamModel, req.Model, nil, "0.000000", "", "", elapsedMs(start, a.now()), clientIP, "error", "upstream_stream_interrupted")
			return ChatResponse{}, ErrUpstream
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			break
		}
		if reason, retry := classifyUpstreamResult(resp.StatusCode, nil); retry {
			a.recordChannelHealth(candidate.ChannelID, false, reason)
			healthRecorded = true
			if attempt+1 < len(candidates) {
				continue
			}
		}
		break
	}
	if resp == nil {
		return ChatResponse{}, ErrUpstream
	}
	_, finalRetryable := classifyUpstreamResult(resp.StatusCode, nil)
	if finalRetryable && len(candidates) > 1 {
		a.logUsage(requestID, auth, &candidate.ChannelID, candidate.UpstreamModel, req.Model, nil, "0.000000", "", "", elapsedMs(start, a.now()), clientIP, "error", fmt.Sprintf("upstream_%d", resp.StatusCode))
		return ChatResponse{}, ErrUpstream
	}
	if req.Stream && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if a.adapter.ParseStream == nil || a.adapter.StreamError == nil {
			resp.Body.Close()
			return ChatResponse{}, ErrInvalidRequest
		}
		releaseReservation = false
		streamOwnsCancel = true
		rateReservationOpen = false
		return ChatResponse{Status: resp.StatusCode, Stream: &completionStream{
			service: a, body: resp.Body, ctx: ctx, requestID: requestID, auth: auth,
			candidate: candidate, publicModel: req.Model, clientIP: clientIP, start: start, reservationID: reservation.ID, rateReservationID: rateReservation.ID, cancel: cancel,
		}}, nil
	}

	if responseBody == nil {
		defer resp.Body.Close()
		responseBody, readErr = io.ReadAll(resp.Body)
	}
	durationMs := elapsedMs(start, a.now())
	if readErr != nil {
		if !errors.Is(readErr, context.Canceled) && !errors.Is(ctx.Err(), context.Canceled) {
			a.recordChannelHealth(candidate.ChannelID, false, domain.FailureUpstreamUnreachable)
		}
		a.logUsage(requestID, auth, &candidate.ChannelID, candidate.UpstreamModel, req.Model, nil, "0.000000", "", "", durationMs, clientIP, "error", "upstream_stream_interrupted")
		return ChatResponse{}, ErrUpstream
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if reason, failure := classifyUpstreamResult(resp.StatusCode, nil); failure && !healthRecorded {
			a.recordChannelHealth(candidate.ChannelID, false, reason)
		}
		a.logUsage(requestID, auth, &candidate.ChannelID, candidate.UpstreamModel, req.Model, nil, "0.000000", "", "", durationMs, clientIP, "error", fmt.Sprintf("upstream_%d", resp.StatusCode))
		return ChatResponse{Status: resp.StatusCode, Body: responseBody}, nil
	}

	if a.adapter.ParseUsage == nil || a.adapter.RewriteResponse == nil {
		return ChatResponse{}, ErrInvalidRequest
	}
	usage := a.adapter.ParseUsage(responseBody)
	if usage == nil {
		a.recordChannelHealth(candidate.ChannelID, false, domain.FailureUpstreamProtocol)
		a.logUsage(requestID, auth, &candidate.ChannelID, candidate.UpstreamModel, req.Model, nil, "0.000000", "", "", durationMs, clientIP, "error", "upstream_usage_missing")
		return ChatResponse{}, ErrUpstream
	}
	a.recordChannelHealth(candidate.ChannelID, true, "")
	cost, inputPrice, outputPrice, err := a.priceFor(candidate.ChannelID, req.Model, usage)
	if err != nil {
		return ChatResponse{}, err
	}

	usageLog := a.usageLogInput(requestID, auth, &candidate.ChannelID, candidate.UpstreamModel, req.Model, usage, cost, inputPrice, outputPrice, durationMs, clientIP, "success", "")
	_, err = a.store.SettleChatCompletion(domain.ChatSettlementInput{
		ReservationID: reservation.ID,
		UserID:        auth.UserID,
		APIKeyID:      auth.KeyID,
		ChannelID:     &candidate.ChannelID,
		Cost:          cost,
		DebitChannel:  candidate.Balance != nil,
		Description:   "chat completion " + requestID,
		UsageLog:      usageLog,
	})
	if err != nil {
		if errors.Is(err, store.ErrInvalid) {
			a.logUsage(requestID, auth, &candidate.ChannelID, candidate.UpstreamModel, req.Model, usage, "0.000000", inputPrice, outputPrice, durationMs, clientIP, "error", "insufficient_balance")
			return ChatResponse{}, ErrInsufficientBalance
		}
		return ChatResponse{}, err
	}
	releaseReservation = false
	_ = a.store.FinalizeRateLimit(context.Background(), rateReservation.ID, int64(usage.TotalTokens))
	rateReservationOpen = false

	// Best-effort: the request already succeeded and was charged, so a
	// last_used_at update failure must not turn it into an error response.
	_ = a.store.UpdateKeyLastUsed(auth.KeyID)

	return ChatResponse{Status: http.StatusOK, Body: a.adapter.RewriteResponse(responseBody, req.Model), Usage: usage}, nil
}

// classifyUpstreamResult decides whether an upstream outcome should count as a
// channel failure. Only transport errors, upstream 429/401/403/402 and 5xx are
// penalised; other client errors (400/404/...) are passed through without
// tripping the breaker, so a bad caller cannot open a healthy channel.
func classifyUpstreamResult(statusCode int, err error) (domain.FailureReason, bool) {
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return domain.FailureUpstreamTimeout, true
		}
		return domain.FailureUpstreamUnreachable, true
	}
	switch statusCode {
	case http.StatusTooManyRequests:
		return domain.FailureUpstream429, true
	case http.StatusUnauthorized:
		return domain.FailureUpstream401, true
	case http.StatusForbidden:
		return domain.FailureUpstream403, true
	case http.StatusPaymentRequired:
		return domain.FailureUpstream402, true
	}
	if statusCode >= 500 && statusCode <= 599 {
		return domain.FailureUpstream5xx, true
	}
	return "", false
}

// recordChannelHealth drives the circuit breaker state machine. It is
// best-effort: a recording failure must never change the response.
func (a *Service) recordChannelHealth(channelID int, success bool, reason domain.FailureReason) {
	_, _ = a.store.RecordChannelAttempt(context.Background(), channelID, success, reason)
}

func (a *Service) priceFor(channelID int, model string, usage *Usage) (string, string, string, error) {
	pricing, err := a.store.GetPricing(channelID, model)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// Known behaviour: a channel+model without pricing is served for
			// free (cost 0). Documented in docs/backend-structure.md; operators
			// should configure pricing for every routable model.
			return "0.000000", "", "", nil
		}
		return "", "", "", err
	}

	inputPrice := pricing.InputPricePer1M
	outputPrice := pricing.OutputPricePer1M
	inputTokens, outputTokens, cachedTokens := 0, 0, 0
	if usage != nil {
		inputTokens = usage.PromptTokens
		outputTokens = usage.CompletionTokens
		cachedTokens = cachedTokenCount(usage)
	}
	cost, err := computeCost(inputPrice, outputPrice, pricing.CachedInputPricePer1M, inputTokens, outputTokens, cachedTokens)
	if err != nil {
		return "", "", "", err
	}
	return cost, inputPrice, outputPrice, nil
}

func (a *Service) logUsage(requestID string, auth *domain.AuthContext, channelID *int, upstreamModel, model string, usage *Usage, cost, inputPrice, outputPrice string, durationMs int, clientIP, status, errorCode string) {
	_, _ = a.store.InsertUsageLog(a.usageLogInput(requestID, auth, channelID, upstreamModel, model, usage, cost, inputPrice, outputPrice, durationMs, clientIP, status, errorCode))
}

func (a *Service) usageLogInput(requestID string, auth *domain.AuthContext, channelID *int, upstreamModel, model string, usage *Usage, cost, inputPrice, outputPrice string, durationMs int, clientIP, status, errorCode string) domain.UsageLogInput {
	userID := auth.UserID
	keyID := auth.KeyID

	input := domain.UsageLogInput{
		RequestID:            requestID,
		UserID:               &userID,
		APIKeyID:             &keyID,
		ChannelID:            channelID,
		Model:                model,
		UpstreamModel:        upstreamModel,
		UnitPriceInputPer1M:  inputPrice,
		UnitPriceOutputPer1M: outputPrice,
		TotalCost:            cost,
		DurationMs:           durationMs,
		Status:               status,
		ErrorCode:            errorCode,
		ClientIP:             clientIP,
	}
	if usage != nil {
		input.InputTokens = usage.PromptTokens
		input.OutputTokens = usage.CompletionTokens
		input.CachedInputTokens = cachedTokenCount(usage)
		input.TotalTokens = usage.TotalTokens
	}
	return input
}

func cachedTokenCount(usage *Usage) int {
	if usage == nil {
		return 0
	}
	return usage.CachedInputTokens
}

func elapsedMs(start, end time.Time) int {
	return int(end.Sub(start).Milliseconds())
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "req_unknown"
	}
	return "req_" + hex.EncodeToString(buf)
}
