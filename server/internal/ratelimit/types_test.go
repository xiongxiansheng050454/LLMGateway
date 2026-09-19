package ratelimit

import "testing"

func TestNormalizeRateLimitRejectsOversizedWindow(t *testing.T) {
	name, target, metric, limit, window, action := "large", "user", "rpm", int64(1), MaxRateLimitWindowSeconds+1, "reject"
	if _, err := NormalizeRateLimit(RateLimitInput{RuleName: &name, TargetType: &target, Metric: &metric, LimitValue: &limit, WindowSeconds: &window, Action: &action}, nil); err == nil {
		t.Fatal("oversized window was accepted")
	}
}

func TestNormalizeRateLimitAcceptsMaximumWindow(t *testing.T) {
	name, target, metric, limit, window, action := "max", "user", "rpm", int64(1), MaxRateLimitWindowSeconds, "reject"
	if _, err := NormalizeRateLimit(RateLimitInput{RuleName: &name, TargetType: &target, Metric: &metric, LimitValue: &limit, WindowSeconds: &window, Action: &action}, nil); err != nil {
		t.Fatal(err)
	}
}
