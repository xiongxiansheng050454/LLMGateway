package domain

import "encoding/json"

type RateLimitRule struct {
	ID            int
	RuleName      string
	TargetType    string
	TargetValue   string
	Metric        string
	LimitValue    int64
	WindowSeconds int
	Action        string
	Priority      int
	Enabled       bool
	Extras        json.RawMessage
}

// RateLimitInput uses pointers so a partial update (for example only
// {"enabled": false}) can be distinguished from omitted fields.
type RateLimitInput struct {
	RuleName      *string         `json:"rule_name"`
	TargetType    *string         `json:"target_type"`
	TargetValue   *string         `json:"target_value"`
	Metric        *string         `json:"metric"`
	LimitValue    *int64          `json:"limit_value"`
	WindowSeconds *int            `json:"window_seconds"`
	Action        *string         `json:"action"`
	Priority      *int            `json:"priority"`
	Enabled       *bool           `json:"enabled"`
	Extras        json.RawMessage `json:"extras"`
}
