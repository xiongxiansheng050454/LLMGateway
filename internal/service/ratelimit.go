package service

import (
	"encoding/json"
	"strconv"
	"time"

	"LLMGateway/internal/domain"
)

// CheckRateLimit enforces enabled rpm rules with the reject action. queue rules
// are intentionally not implemented (recorded as a known limitation).
//
// A key's rate_limit_overrides.rpm takes precedence over matching rules.
func (p *Proxy) CheckRateLimit(auth *domain.AuthContext, model string) error {
	since := p.now().Add(-time.Minute).UTC().Format(time.RFC3339)

	if override := rpmOverride(auth.RateLimitOverrides); override > 0 {
		count, err := p.store.CountRequestsSince(auth.UserID, &auth.KeyID, since)
		if err != nil {
			return err
		}
		if int64(count) >= override {
			return ErrRateLimited
		}
	}

	enabled := true
	result, err := p.store.ListRateLimits(&enabled, 1, 1000)
	if err != nil {
		return err
	}
	for _, item := range result.List {
		rule, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if toString(rule["metric"]) != "rpm" || toString(rule["action"]) != "reject" {
			continue
		}
		if !matchesTarget(rule, auth, model) {
			continue
		}
		limit := toInt64(rule["limit_value"])
		if limit <= 0 {
			continue
		}
		count, err := p.store.CountRequestsSince(auth.UserID, &auth.KeyID, since)
		if err != nil {
			return err
		}
		if int64(count) >= limit {
			return ErrRateLimited
		}
	}
	return nil
}

// matchesTarget decides whether a rule applies to the current request.
//
// Known limitation: rpm counting is per user/key, not per model or channel, so
// a model-scoped rule counts all of that user's requests. channel scoping is not
// evaluated because the channel is not chosen until after the limit check.
func matchesTarget(rule map[string]any, auth *domain.AuthContext, model string) bool {
	targetType := toString(rule["target_type"])
	targetValue := toString(rule["target_value"])

	switch targetType {
	case "global":
		return true
	case "user":
		return targetValue == "*" || targetValue == strconv.Itoa(auth.UserID)
	case "api_key":
		return targetValue == "*" || targetValue == strconv.Itoa(auth.KeyID)
	case "model":
		return targetValue == "*" || targetValue == model
	default:
		// channel scoping is not known before routing and is not enforced here.
		return false
	}
}

func rpmOverride(raw json.RawMessage) int64 {
	if len(raw) == 0 {
		return 0
	}
	var overrides struct {
		RPM int64 `json:"rpm"`
	}
	if err := json.Unmarshal(raw, &overrides); err != nil {
		return 0
	}
	return overrides.RPM
}
