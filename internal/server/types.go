package server

type adminResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type listResponse struct {
	List  []any `json:"list"`
	Total int   `json:"total"`
}

type channel struct {
	ID       int
	Name     string
	BaseURL  string
	APIKey   string
	AuthType string
	Status   int
	Weight   int
	Priority int
	Balance  *string
}

type channelModel struct {
	ID            int    `json:"id"`
	ModelName     string `json:"model_name"`
	UpstreamModel string `json:"upstream_model"`
	Enabled       bool   `json:"enabled"`
}

type pricing struct {
	ID                    int
	ChannelID             int
	ModelName             string
	InputPricePer1M       string
	OutputPricePer1M      string
	CachedInputPricePer1M string
	Currency              string
}

type channelInput struct {
	Name     string  `json:"name"`
	BaseURL  string  `json:"base_url"`
	APIKey   string  `json:"api_key"`
	AuthType string  `json:"auth_type"`
	Status   int     `json:"status"`
	Weight   int     `json:"weight"`
	Priority int     `json:"priority"`
	Balance  *string `json:"balance"`
}

type pricingInput struct {
	ChannelID             int    `json:"channel_id"`
	ModelName             string `json:"model_name"`
	InputPricePer1M       string `json:"input_price_per_1m"`
	OutputPricePer1M      string `json:"output_price_per_1m"`
	CachedInputPricePer1M string `json:"cached_input_price_per_1m"`
	Currency              string `json:"currency"`
}

type deletePricingInput struct {
	ChannelID int    `json:"channel_id"`
	ModelName string `json:"model_name"`
}
