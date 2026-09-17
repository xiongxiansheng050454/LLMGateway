package proxy

import (
	"encoding/json"
	"strconv"
	"time"

	"LLMGateway/server/internal/domain"
)

// checkRateLimit enforces enabled rpm rules with the reject action. queue rules
// are intentionally not implemented (recorded as a known limitation).
//
// A key's rate_limit_overrides.rpm takes precedence over matching rules.
func (a *Service) checkRateLimit(auth *domain.AuthContext, model string) error {
	since := a.now().Add(-time.Minute).UTC().Format(time.RFC3339)

	if override := rpmOverride(auth.RateLimitOverrides); override > 0 {
		count, err := a.store.CountRequestsSince(domain.UsageCountFilter{UserID: auth.UserID, APIKeyID: &auth.KeyID, Since: since})
		if err != nil {
			return err
		}
		if int64(count) >= override {
			return ErrRateLimited
		}
	}

	enabled := true
	result, err := a.store.ListRateLimits(&enabled, 1, 1000)
	if err != nil {
		return err
	}
	for _, item := range result.List {
		if item.Metric != "rpm" || item.Action != "reject" {
			continue
		}
		if item.TargetType == "channel" {
			continue
		}
		if !matchesTarget(item, auth, model) {
			continue
		}
		limit := item.LimitValue
		if limit <= 0 {
			continue
		}
		filter := domain.UsageCountFilter{UserID: auth.UserID, APIKeyID: &auth.KeyID, Since: since}
		if item.TargetType == "model" {
			filter.Model = model
		}
		count, err := a.store.CountRequestsSince(filter)
		if err != nil {
			return err
		}
		if int64(count) >= limit {
			return ErrRateLimited
		}
	}
	return nil
}

func (a *Service) checkChannelRateLimit(auth *domain.AuthContext, model string, channelID int) error {
	since := a.now().Add(-time.Minute).UTC().Format(time.RFC3339)
	enabled := true
	result, err := a.store.ListRateLimits(&enabled, 1, 1000)
	if err != nil {
		return err
	}
	for _, item := range result.List {
		if item.Metric != "rpm" || item.Action != "reject" || item.TargetType != "channel" {
			continue
		}
		if item.TargetValue != "*" && item.TargetValue != strconv.Itoa(channelID) {
			continue
		}
		if item.LimitValue <= 0 {
			continue
		}
		count, err := a.store.CountRequestsSince(domain.UsageCountFilter{UserID: auth.UserID, APIKeyID: &auth.KeyID, Since: since, ChannelID: &channelID})
		if err != nil {
			return err
		}
		if int64(count) >= item.LimitValue {
			return ErrRateLimited
		}
	}
	return nil
}

// matchesTarget decides whether a rule applies to the current request.
func matchesTarget(rule domain.RateLimitRuleDTO, auth *domain.AuthContext, model string) bool {
	switch rule.TargetType {
	case "global":
		return true
	case "user":
		return rule.TargetValue == "*" || rule.TargetValue == strconv.Itoa(auth.UserID)
	case "api_key":
		return rule.TargetValue == "*" || rule.TargetValue == strconv.Itoa(auth.KeyID)
	case "model":
		return rule.TargetValue == "*" || rule.TargetValue == model
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
