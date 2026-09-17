package openai

// Package openai contains the OpenAI wire adapter types owned by the proxy
// business module.

// OpenAIModel is a single entry in GET /v1/models.
type OpenAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type OpenAIModelList struct {
	Object string        `json:"object"`
	Data   []OpenAIModel `json:"data"`
}

type OpenAIError struct {
	Error OpenAIErrorBody `json:"error"`
}

type OpenAIErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// ChatCompletionRequest carries only the fields the gateway needs to inspect.
// The original body is forwarded upstream so unknown fields are preserved.
type ChatCompletionRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

// ChatCompletionUsage mirrors the OpenAI usage object.
type ChatCompletionUsage struct {
	PromptTokens        int `json:"prompt_tokens"`
	CompletionTokens    int `json:"completion_tokens"`
	TotalTokens         int `json:"total_tokens"`
	PromptTokensDetails *struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
}

// ChatCompletionResponse is used to read usage and rewrite the model name while
// preserving the remaining upstream fields.
type ChatCompletionResponse struct {
	Model string               `json:"model"`
	Usage *ChatCompletionUsage `json:"usage"`
}
