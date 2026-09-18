package proxy

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"LLMGateway/server/internal/domain"
)

type rateLimitOverrides struct{ RPM, TPM, RPD, TPD, Concurrency int64 }

func parseRateLimitOverrides(raw json.RawMessage) rateLimitOverrides {
	var value rateLimitOverrides
	_ = json.Unmarshal(raw, &value)
	return value
}

func applicableOverride(overrides rateLimitOverrides, rule domain.RateLimitRuleDTO) (int64, bool) {
	if rule.TargetType != "api_key" {
		return 0, false
	}
	switch rule.Metric {
	case "rpm":
		return overrides.RPM, overrides.RPM > 0
	case "tpm":
		return overrides.TPM, overrides.TPM > 0
	case "rpd":
		return overrides.RPD, overrides.RPD > 0
	case "tpd":
		return overrides.TPD, overrides.TPD > 0
	case "concurrency":
		return overrides.Concurrency, overrides.Concurrency > 0
	}
	return 0, false
}

func slidingWindowCount(previous, current, windowSeconds int64, now, bucketStart time.Time) int64 {
	if windowSeconds <= 0 {
		return current
	}
	elapsed := now.Sub(bucketStart).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > float64(windowSeconds) {
		elapsed = float64(windowSeconds)
	}
	return (previous*(windowSeconds-int64(elapsed)) + current*int64(elapsed)) / windowSeconds
}

// checkRateLimit enforces enabled rpm rules with the reject action. queue rules
// are intentionally not implemented (recorded as a known limitation).
//
// A key's rate_limit_overrides.rpm takes precedence over matching rules.
func (a *Service) checkRateLimit(auth *domain.AuthContext, model string, estimatedTokens *int64) error {
	since := a.now().Add(-time.Minute).UTC().Format(time.RFC3339)

	if override := parseRateLimitOverrides(auth.RateLimitOverrides).RPM; override > 0 {
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
		if item.Action != "reject" {
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
		if override, ok := applicableOverride(parseRateLimitOverrides(auth.RateLimitOverrides), item); ok {
			limit = override
		}
		var count int64
		if item.Metric == "rpm" || item.Metric == "rpd" {
			since := metricSince(a.now(), item.Metric, item.WindowSeconds)
			filter.Since = since
			current, countErr := a.store.CountRequestsSince(filter)
			if countErr != nil {
				return countErr
			}
			count = int64(current)
		} else if item.Metric == "tpm" || item.Metric == "tpd" {
			if estimatedTokens == nil {
				return ErrRateLimited
			}
			since := metricSince(a.now(), item.Metric, item.WindowSeconds)
			tokenCount, countErr := a.store.CountTokensSince(domain.TokenCountFilter{UserID: auth.UserID, APIKeyID: &auth.KeyID, Model: filter.Model, Since: since})
			if countErr != nil {
				return countErr
			}
			count = tokenCount + *estimatedTokens
		} else if item.Metric == "concurrency" {
			current, countErr := a.store.CountActiveRateLimitReservations(context.Background(), auth.UserID, &auth.KeyID, filter.Model, nil)
			if countErr != nil {
				return countErr
			}
			count = current + 1
		} else {
			continue
		}
		if count >= limit {
			return ErrRateLimited
		}
	}
	return nil
}

func metricSince(now time.Time, metric string, windowSeconds int) string {
	now = now.UTC()
	if metric == "rpd" || metric == "tpd" {
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	}
	return now.Add(-time.Duration(windowSeconds) * time.Second).Format(time.RFC3339)
}

func (a *Service) checkChannelRateLimit(auth *domain.AuthContext, model string, channelID int, estimatedTokens int64) error {
	enabled := true
	result, err := a.store.ListRateLimits(&enabled, 1, 1000)
	if err != nil {
		return err
	}
	for _, item := range result.List {
		if item.Action != "reject" || item.TargetType != "channel" {
			continue
		}
		if item.TargetValue != "*" && item.TargetValue != strconv.Itoa(channelID) {
			continue
		}
		if item.LimitValue <= 0 {
			continue
		}
		var count int64
		var err error
		if item.Metric == "rpm" || item.Metric == "rpd" {
			current, countErr := a.store.CountRequestsSince(domain.UsageCountFilter{UserID: auth.UserID, APIKeyID: &auth.KeyID, Since: metricSince(a.now(), item.Metric, item.WindowSeconds), ChannelID: &channelID})
			err = countErr
			count = int64(current)
		} else if item.Metric == "tpm" || item.Metric == "tpd" {
			current, countErr := a.store.CountTokensSince(domain.TokenCountFilter{UserID: auth.UserID, APIKeyID: &auth.KeyID, Since: metricSince(a.now(), item.Metric, item.WindowSeconds), Model: model, ChannelID: &channelID})
			err = countErr
			count = current + estimatedTokens
		} else if item.Metric == "concurrency" {
			current, countErr := a.store.CountActiveRateLimitReservations(context.Background(), auth.UserID, &auth.KeyID, model, &channelID)
			err = countErr
			count = current + 1
		} else {
			continue
		}
		if err != nil {
			return err
		}
		if count >= item.LimitValue {
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
	return parseRateLimitOverrides(raw).RPM
}
