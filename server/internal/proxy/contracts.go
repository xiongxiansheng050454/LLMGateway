package proxy

// ChatRequest is the protocol-neutral input needed by proxy orchestration.
type ChatRequest struct {
	Model  string
	Stream bool
	Body   []byte
}

// Usage contains the token counts needed for settlement and audit logging.
type Usage struct {
	PromptTokens      int
	CompletionTokens  int
	TotalTokens       int
	CachedInputTokens int
}

// Model is a public model visible to an authenticated caller.
type Model struct {
	ID      string
	Created int64
	OwnedBy string
}

// ModelList is the protocol-neutral result of model discovery.
type ModelList struct {
	Models []Model
}

// ChatResponse carries an upstream response without protocol-specific DTOs.
type ChatResponse struct {
	Status int
	Body   []byte
	Usage  *Usage
}

// ProtocolAdapter contains the small protocol seam needed by proxy business
// orchestration. The proxy package owns the contract; adapters own wire rules.
type ProtocolAdapter struct {
	RewriteRequest  func([]byte, string) ([]byte, error)
	ParseUsage      func([]byte) *Usage
	RewriteResponse func([]byte, string) []byte
}
