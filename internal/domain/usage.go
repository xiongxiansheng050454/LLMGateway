package domain

type UsageLog struct {
	ID                   int
	RequestID            string
	UserID               *int
	APIKeyID             *int
	ChannelID            *int
	ChannelName          string
	Model                string
	UpstreamModel        string
	InputTokens          int
	OutputTokens         int
	CachedInputTokens    int
	TotalTokens          int
	UnitPriceInputPer1M  string
	UnitPriceOutputPer1M string
	TotalCost            string
	DurationMs           int
	TTFTMs               *int
	Status               string
	ErrorCode            string
	ClientIP             string
	CreatedAt            string
}

type UsageLogFilter struct {
	UserID    *int
	ChannelID *int
	Model     string
	Status    string
	StartTime string
	EndTime   string
	Page      int
	PageSize  int
}

// UsageLogInput is the write model reused by the downstream proxy (#6).
type UsageLogInput struct {
	RequestID            string
	UserID               *int
	APIKeyID             *int
	ChannelID            *int
	Model                string
	UpstreamModel        string
	InputTokens          int
	OutputTokens         int
	CachedInputTokens    int
	TotalTokens          int
	UnitPriceInputPer1M  string
	UnitPriceOutputPer1M string
	TotalCost            string
	DurationMs           int
	TTFTMs               *int
	Status               string
	ErrorCode            string
	ClientIP             string
}
