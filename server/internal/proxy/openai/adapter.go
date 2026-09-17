package openai

import (
	"encoding/json"

	"LLMGateway/server/internal/proxy"
)

// Adapter provides OpenAI wire conversion at the protocol boundary.
func Adapter() proxy.ProtocolAdapter {
	return proxy.ProtocolAdapter{
		RewriteRequest:  rewriteRequest,
		ParseUsage:      parseUsage,
		RewriteResponse: rewriteResponse,
	}
}

func ParseRequest(body []byte) (proxy.ChatRequest, error) {
	var request ChatCompletionRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return proxy.ChatRequest{}, err
	}
	return proxy.ChatRequest{Model: request.Model, Stream: request.Stream, Body: body}, nil
}

func rewriteRequest(body []byte, upstreamModel string) ([]byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	payload["model"] = upstreamModel
	return json.Marshal(payload)
}

func parseUsage(body []byte) *proxy.Usage {
	var response ChatCompletionResponse
	if err := json.Unmarshal(body, &response); err != nil || response.Usage == nil {
		return nil
	}
	usage := &proxy.Usage{
		PromptTokens:     response.Usage.PromptTokens,
		CompletionTokens: response.Usage.CompletionTokens,
		TotalTokens:      response.Usage.TotalTokens,
	}
	if response.Usage.PromptTokensDetails != nil {
		usage.CachedInputTokens = response.Usage.PromptTokensDetails.CachedTokens
	}
	return usage
}

func rewriteResponse(body []byte, publicModel string) []byte {
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
