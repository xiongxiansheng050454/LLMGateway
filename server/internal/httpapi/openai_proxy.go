package httpapi

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/money"
	openaiwire "LLMGateway/server/internal/protocol/openai"
	"LLMGateway/server/internal/store"
)

// models returns the OpenAI-style model list visible to the key, filtered by
// its permissions.
func (a *Server) models(auth *domain.AuthContext) (openaiwire.OpenAIModelList, error) {
	result, err := a.store.ListCatalogModels(true)
	if err != nil {
		return openaiwire.OpenAIModelList{}, err
	}

	created := a.now().Unix()
	seen := map[string]bool{}
	data := []openaiwire.OpenAIModel{}
	for _, item := range result.List {
		name := item.ModelName
		if name == "" || seen[name] || !allowModel(auth, name) {
			continue
		}
		seen[name] = true
		data = append(data, openaiwire.OpenAIModel{ID: name, Object: "model", Created: created, OwnedBy: "llmgateway"})
	}
	return openaiwire.OpenAIModelList{Object: "list", Data: data}, nil
}

// chatCompletions proxies a non-streaming chat completion request. It returns
// the HTTP status and body to send downstream. A non-nil error is a
// pre-flight/transport failure the handler maps to an OpenAI error.
func (a *Server) chatCompletions(auth *domain.AuthContext, body []byte, clientIP string) (int, []byte, error) {
	var req openaiwire.ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return 0, nil, ErrInvalidRequest
	}
	if req.Model == "" {
		return 0, nil, ErrInvalidRequest
	}
	if req.Stream {
		return 0, nil, ErrStreamingUnsupported
	}
	if !allowModel(auth, req.Model) {
		return 0, nil, ErrForbidden
	}
	balance, err := money.Parse6(auth.AvailableBalance)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: invalid balance", store.ErrInvalid)
	}
	if balance.Cmp(0) <= 0 {
		return 0, nil, ErrInsufficientBalance
	}

	requestID := newRequestID()
	start := a.now()
	if err := a.checkRateLimit(auth, req.Model); err != nil {
		if errors.Is(err, ErrRateLimited) {
			a.logUsage(requestID, auth, nil, "", req.Model, nil, "0.000000", "", "", elapsedMs(start, a.now()), clientIP, "error", "rate_limited")
		}
		return 0, nil, err
	}

	candidate, err := a.selectChannel(req.Model)
	if err != nil {
		if errors.Is(err, ErrNoHealthyChannel) {
			// Degraded: every candidate is tripped open or there is no mapping.
			a.logUsage(requestID, auth, nil, "", req.Model, nil, "0.000000", "", "", elapsedMs(start, a.now()), clientIP, "error", "no_healthy_channel")
		}
		return 0, nil, err
	}
	if err := a.checkChannelRateLimit(auth, req.Model, candidate.ChannelID); err != nil {
		if errors.Is(err, ErrRateLimited) {
			a.logUsage(requestID, auth, &candidate.ChannelID, candidate.UpstreamModel, req.Model, nil, "0.000000", "", "", elapsedMs(start, a.now()), clientIP, "error", "rate_limited")
		}
		return 0, nil, err
	}

	secret, err := a.store.GetChannelSecret(candidate.ChannelID)
	if err != nil {
		return 0, nil, err
	}

	upstreamBody, err := rewriteModel(body, candidate.UpstreamModel)
	if err != nil {
		return 0, nil, ErrInvalidRequest
	}

	httpReq, err := http.NewRequest(http.MethodPost, strings.TrimRight(secret.BaseURL, "/")+"/v1/chat/completions", bytes.NewReader(upstreamBody))
	if err != nil {
		return 0, nil, ErrUpstream
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if secret.AuthType == "bearer" && secret.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+secret.APIKey)
	}

	resp, err := a.client.Do(httpReq)
	if err != nil {
		if reason, failure := classifyUpstreamResult(0, err); failure {
			a.recordChannelHealth(candidate.ChannelID, false, reason)
		}
		a.logUsage(requestID, auth, &candidate.ChannelID, candidate.UpstreamModel, req.Model, nil, "0.000000", "", "", elapsedMs(start, a.now()), clientIP, "error", "upstream_unreachable")
		return 0, nil, ErrUpstream
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(resp.Body)
	durationMs := elapsedMs(start, a.now())

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if reason, failure := classifyUpstreamResult(resp.StatusCode, nil); failure {
			a.recordChannelHealth(candidate.ChannelID, false, reason)
		}
		a.logUsage(requestID, auth, &candidate.ChannelID, candidate.UpstreamModel, req.Model, nil, "0.000000", "", "", durationMs, clientIP, "error", fmt.Sprintf("upstream_%d", resp.StatusCode))
		return resp.StatusCode, responseBody, nil
	}

	a.recordChannelHealth(candidate.ChannelID, true, "")

	usage := parseUsage(responseBody)
	cost, inputPrice, outputPrice, err := a.priceFor(candidate.ChannelID, req.Model, usage)
	if err != nil {
		return 0, nil, err
	}

	usageLog := a.usageLogInput(requestID, auth, &candidate.ChannelID, candidate.UpstreamModel, req.Model, usage, cost, inputPrice, outputPrice, durationMs, clientIP, "success", "")
	_, err = a.store.SettleChatCompletion(domain.ChatSettlementInput{
		UserID:       auth.UserID,
		ChannelID:    &candidate.ChannelID,
		Cost:         cost,
		DebitChannel: candidate.Balance != nil,
		Description:  "chat completion " + requestID,
		UsageLog:     usageLog,
	})
	if err != nil {
		if errors.Is(err, store.ErrInvalid) {
			a.logUsage(requestID, auth, &candidate.ChannelID, candidate.UpstreamModel, req.Model, usage, "0.000000", inputPrice, outputPrice, durationMs, clientIP, "error", "insufficient_balance")
			return 0, nil, ErrInsufficientBalance
		}
		return 0, nil, err
	}

	// Best-effort: the request already succeeded and was charged, so a
	// last_used_at update failure must not turn it into an error response.
	_ = a.store.UpdateKeyLastUsed(auth.KeyID)

	return http.StatusOK, rewriteResponseModel(responseBody, req.Model), nil
}

// classifyUpstreamResult decides whether an upstream outcome should count as a
// channel failure. Only transport errors, upstream 429/401/403/402 and 5xx are
// penalised; other client errors (400/404/...) are passed through without
// tripping the breaker, so a bad caller cannot open a healthy channel.
func classifyUpstreamResult(statusCode int, err error) (domain.FailureReason, bool) {
	if err != nil {
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
func (a *Server) recordChannelHealth(channelID int, success bool, reason domain.FailureReason) {
	if success {
		_, _ = a.store.RecordChannelSuccess(channelID)
		return
	}
	_, _ = a.store.RecordChannelFailure(channelID, reason)
}

func (a *Server) priceFor(channelID int, model string, usage *openaiwire.ChatCompletionUsage) (string, string, string, error) {
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

func (a *Server) logUsage(requestID string, auth *domain.AuthContext, channelID *int, upstreamModel, model string, usage *openaiwire.ChatCompletionUsage, cost, inputPrice, outputPrice string, durationMs int, clientIP, status, errorCode string) {
	_, _ = a.store.InsertUsageLog(a.usageLogInput(requestID, auth, channelID, upstreamModel, model, usage, cost, inputPrice, outputPrice, durationMs, clientIP, status, errorCode))
}

func (a *Server) usageLogInput(requestID string, auth *domain.AuthContext, channelID *int, upstreamModel, model string, usage *openaiwire.ChatCompletionUsage, cost, inputPrice, outputPrice string, durationMs int, clientIP, status, errorCode string) domain.UsageLogInput {
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

func cachedTokenCount(usage *openaiwire.ChatCompletionUsage) int {
	if usage == nil || usage.PromptTokensDetails == nil {
		return 0
	}
	return usage.PromptTokensDetails.CachedTokens
}

func parseUsage(body []byte) *openaiwire.ChatCompletionUsage {
	var parsed openaiwire.ChatCompletionResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil
	}
	return parsed.Usage
}

func rewriteModel(body []byte, upstreamModel string) ([]byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	payload["model"] = upstreamModel
	return json.Marshal(payload)
}

func rewriteResponseModel(body []byte, publicModel string) []byte {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	payload["model"] = publicModel
	rewritten, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return rewritten
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
