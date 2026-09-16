package service

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

	"LLMGateway/internal/domain"
	"LLMGateway/internal/money"
	"LLMGateway/internal/store"
)

// Models returns the OpenAI-style model list visible to the key, filtered by
// its permissions.
func (p *Proxy) Models(auth *domain.AuthContext) (domain.OpenAIModelList, error) {
	result, err := p.store.ListCatalogModels(true)
	if err != nil {
		return domain.OpenAIModelList{}, err
	}

	created := p.now().Unix()
	seen := map[string]bool{}
	data := []domain.OpenAIModel{}
	for _, item := range result.List {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := toString(entry["model_name"])
		if name == "" || seen[name] || !allowModel(auth, name) {
			continue
		}
		seen[name] = true
		data = append(data, domain.OpenAIModel{ID: name, Object: "model", Created: created, OwnedBy: "llmgateway"})
	}
	return domain.OpenAIModelList{Object: "list", Data: data}, nil
}

// ChatCompletions proxies a non-streaming chat completion request. It returns
// the HTTP status and body to send downstream. A non-nil error is a
// pre-flight/transport failure the handler maps to an OpenAI error.
func (p *Proxy) ChatCompletions(auth *domain.AuthContext, body []byte, clientIP string) (int, []byte, error) {
	var req domain.ChatCompletionRequest
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
	if err := p.CheckRateLimit(auth, req.Model); err != nil {
		return 0, nil, err
	}

	balance, err := money.Parse6(auth.AvailableBalance)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: invalid balance", store.ErrInvalid)
	}
	if balance.Cmp(0) <= 0 {
		return 0, nil, ErrInsufficientBalance
	}

	candidate, err := p.SelectChannel(req.Model)
	if err != nil {
		return 0, nil, err
	}

	secret, err := p.store.GetChannelSecret(candidate.ChannelID)
	if err != nil {
		return 0, nil, err
	}

	upstreamBody, err := rewriteModel(body, candidate.UpstreamModel)
	if err != nil {
		return 0, nil, ErrInvalidRequest
	}

	requestID := newRequestID()
	start := p.now()

	httpReq, err := http.NewRequest(http.MethodPost, strings.TrimRight(secret.BaseURL, "/")+"/v1/chat/completions", bytes.NewReader(upstreamBody))
	if err != nil {
		return 0, nil, ErrUpstream
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if secret.AuthType == "bearer" && secret.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+secret.APIKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		p.logUsage(requestID, auth, candidate, req.Model, nil, "0.000000", "", "", elapsedMs(start, p.now()), clientIP, "error", "upstream_unreachable")
		return 0, nil, ErrUpstream
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(resp.Body)
	durationMs := elapsedMs(start, p.now())

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		p.logUsage(requestID, auth, candidate, req.Model, nil, "0.000000", "", "", durationMs, clientIP, "error", fmt.Sprintf("upstream_%d", resp.StatusCode))
		return resp.StatusCode, responseBody, nil
	}

	usage := parseUsage(responseBody)
	cost, inputPrice, outputPrice, err := p.priceFor(candidate.ChannelID, req.Model, usage)
	if err != nil {
		return 0, nil, err
	}

	if cost != "0.000000" {
		if _, err := p.store.DebitUserBalance(auth.UserID, cost, "chat completion "+requestID); err != nil {
			if errors.Is(err, store.ErrInvalid) {
				p.logUsage(requestID, auth, candidate, req.Model, usage, "0.000000", inputPrice, outputPrice, durationMs, clientIP, "error", "insufficient_balance")
				return 0, nil, ErrInsufficientBalance
			}
			return 0, nil, err
		}
		if candidate.Balance != nil {
			// Channel balance is best-effort; the user has already been charged.
			_, _ = p.store.UpdateChannelBalance(candidate.ChannelID, "", "-"+cost)
		}
	}

	p.logUsage(requestID, auth, candidate, req.Model, usage, cost, inputPrice, outputPrice, durationMs, clientIP, "success", "")

	// Best-effort: the request already succeeded and was charged, so a
	// last_used_at update failure must not turn it into an error response.
	_ = p.store.UpdateKeyLastUsed(auth.KeyID)

	return http.StatusOK, rewriteResponseModel(responseBody, req.Model), nil
}

func (p *Proxy) priceFor(channelID int, model string, usage *domain.ChatCompletionUsage) (string, string, string, error) {
	pricing, err := p.store.GetPricing(channelID, model)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// Known behaviour: a channel+model without pricing is served for
			// free (cost 0). Documented in docs/backend-structure.md; operators
			// should configure pricing for every routable model.
			return "0.000000", "", "", nil
		}
		return "", "", "", err
	}

	inputPrice := toString(pricing["input_price_per_1m"])
	outputPrice := toString(pricing["output_price_per_1m"])
	inputTokens, outputTokens, cachedTokens := 0, 0, 0
	if usage != nil {
		inputTokens = usage.PromptTokens
		outputTokens = usage.CompletionTokens
		cachedTokens = cachedTokenCount(usage)
	}
	cost, err := computeCost(inputPrice, outputPrice, toString(pricing["cached_input_price_per_1m"]), inputTokens, outputTokens, cachedTokens)
	if err != nil {
		return "", "", "", err
	}
	return cost, inputPrice, outputPrice, nil
}

func (p *Proxy) logUsage(requestID string, auth *domain.AuthContext, candidate RouteCandidate, model string, usage *domain.ChatCompletionUsage, cost, inputPrice, outputPrice string, durationMs int, clientIP, status, errorCode string) {
	userID := auth.UserID
	keyID := auth.KeyID
	channelID := candidate.ChannelID

	input := domain.UsageLogInput{
		RequestID:            requestID,
		UserID:               &userID,
		APIKeyID:             &keyID,
		ChannelID:            &channelID,
		Model:                model,
		UpstreamModel:        candidate.UpstreamModel,
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
	_, _ = p.store.InsertUsageLog(input)
}

func cachedTokenCount(usage *domain.ChatCompletionUsage) int {
	if usage == nil || usage.PromptTokensDetails == nil {
		return 0
	}
	return usage.PromptTokensDetails.CachedTokens
}

func parseUsage(body []byte) *domain.ChatCompletionUsage {
	var parsed domain.ChatCompletionResponse
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

func toInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func toInt64(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int32:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

func toString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(typed)
	}
}
