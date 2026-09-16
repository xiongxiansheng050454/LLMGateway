package store

import (
	"encoding/json"
	"fmt"
	"strings"

	"LLMGateway/server/internal/domain"
)

// RateLimitStore manages rate limit rules. This issue only stores rules; it does
// not enforce them at request time.
type RateLimitStore interface {
	ListRateLimits(enabled *bool, page, pageSize int) (domain.ListResponse, error)
	CreateRateLimit(domain.RateLimitInput) (map[string]any, error)
	UpdateRateLimit(int, domain.RateLimitInput) (map[string]any, error)
	DeleteRateLimit(int) error
}

var (
	validTargetTypes = map[string]bool{"global": true, "user": true, "api_key": true, "model": true, "channel": true}
	validMetrics     = map[string]bool{"rpm": true, "tpm": true, "rpd": true, "tpd": true, "concurrency": true}
	validActions     = map[string]bool{"reject": true, "queue": true}
)

// NormalizeRateLimit merges a partial input over an existing rule (or defaults
// for a new rule) and validates the result. It is shared by the memory and
// PostgreSQL implementations so both accept and reject the same values.
func NormalizeRateLimit(in domain.RateLimitInput, existing *domain.RateLimitRule) (domain.RateLimitRule, error) {
	rule := domain.RateLimitRule{TargetValue: "*", Priority: 100, Enabled: true, Extras: json.RawMessage(`{}`)}
	if existing != nil {
		rule = *existing
	}

	if in.RuleName != nil {
		rule.RuleName = *in.RuleName
	}
	if in.TargetType != nil {
		rule.TargetType = *in.TargetType
	}
	if in.TargetValue != nil {
		rule.TargetValue = *in.TargetValue
	}
	if in.Metric != nil {
		rule.Metric = *in.Metric
	}
	if in.LimitValue != nil {
		rule.LimitValue = *in.LimitValue
	}
	if in.WindowSeconds != nil {
		rule.WindowSeconds = *in.WindowSeconds
	}
	if in.Action != nil {
		rule.Action = *in.Action
	}
	if in.Priority != nil {
		rule.Priority = *in.Priority
	}
	if in.Enabled != nil {
		rule.Enabled = *in.Enabled
	}
	if len(in.Extras) > 0 {
		rule.Extras = in.Extras
	}

	if rule.TargetValue == "" {
		rule.TargetValue = "*"
	}
	if len(rule.Extras) == 0 {
		rule.Extras = json.RawMessage(`{}`)
	}

	switch {
	case strings.TrimSpace(rule.RuleName) == "":
		return rule, fmt.Errorf("%w: rule_name is required", ErrInvalid)
	case !validTargetTypes[rule.TargetType]:
		return rule, fmt.Errorf("%w: invalid target_type", ErrInvalid)
	case !validMetrics[rule.Metric]:
		return rule, fmt.Errorf("%w: invalid metric", ErrInvalid)
	case !validActions[rule.Action]:
		return rule, fmt.Errorf("%w: invalid action", ErrInvalid)
	case rule.LimitValue <= 0:
		return rule, fmt.Errorf("%w: limit_value must be positive", ErrInvalid)
	case rule.WindowSeconds <= 0:
		return rule, fmt.Errorf("%w: window_seconds must be positive", ErrInvalid)
	}
	return rule, nil
}
