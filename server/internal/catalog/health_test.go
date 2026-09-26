package catalog

import (
	"testing"
	"time"
)

var breakerTestNow = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)

func TestApplyChannelFailureOpensAtThreshold(t *testing.T) {
	cfg := ChannelBreakerConfig{FailureThreshold: 3, Cooldown: 30 * time.Second}
	health := NewChannelHealth(1)
	for i := 1; i <= 2; i++ {
		health = ApplyChannelFailure(health, FailureUpstream5xx, ChannelHealthWindow{}, breakerTestNow, cfg)
		if health.State != HealthClosed {
			t.Fatalf("after %d failures state = %s, want closed", i, health.State)
		}
	}
	health = ApplyChannelFailure(health, FailureUpstream5xx, ChannelHealthWindow{}, breakerTestNow, cfg)
	if health.State != HealthOpen || health.OpenedAt == nil {
		t.Fatalf("health = %+v, want open with timestamp", health)
	}
	if health.ConsecutiveFailures != 3 || health.FailureCount != 3 {
		t.Fatalf("counters = %d/%d, want 3/3", health.ConsecutiveFailures, health.FailureCount)
	}
}

func TestDeterministicFailureOpensImmediately(t *testing.T) {
	cfg := ChannelBreakerConfig{FailureThreshold: 5, Cooldown: 30 * time.Second}
	if got := ApplyChannelFailure(NewChannelHealth(1), FailureUpstream401, ChannelHealthWindow{}, breakerTestNow, cfg); got.State != HealthOpen {
		t.Fatalf("state = %s, want open", got.State)
	}
}

func TestEvaluateTransitionsToHalfOpenAfterCooldown(t *testing.T) {
	cfg := ChannelBreakerConfig{FailureThreshold: 5, Cooldown: 30 * time.Second}
	opened := breakerTestNow.Format(time.RFC3339)
	health := ChannelHealth{ChannelID: 1, State: HealthOpen, OpenedAt: &opened, ConsecutiveFailures: 5}
	if got := EvaluateChannelHealth(health, breakerTestNow.Add(29*time.Second), cfg); got.State != HealthOpen {
		t.Fatalf("before cooldown state = %s", got.State)
	}
	got := EvaluateChannelHealth(health, breakerTestNow.Add(30*time.Second), cfg)
	if got.State != HealthHalfOpen || got.ConsecutiveFailures != 0 || got.OpenedAt != nil {
		t.Fatalf("half-open health = %+v", got)
	}
}

func TestHalfOpenSuccessClosesAndFailureReopens(t *testing.T) {
	cfg := ChannelBreakerConfig{FailureThreshold: 5, Cooldown: 30 * time.Second}
	halfOpen := ChannelHealth{ChannelID: 1, State: HealthHalfOpen, FailureCount: 5}
	if got := ApplyChannelSuccess(halfOpen, breakerTestNow); got.State != HealthClosed || got.ConsecutiveFailures != 0 || got.OpenedAt != nil {
		t.Fatalf("closed health = %+v", got)
	}
	if got := ApplyChannelFailure(halfOpen, FailureUpstream5xx, ChannelHealthWindow{}, breakerTestNow, cfg); got.State != HealthOpen {
		t.Fatalf("reopened state = %s", got.State)
	}
}

func TestBreakerWindowOpensOnlyAfterMinimumSamplesAndIntegerThreshold(t *testing.T) {
	cfg := ChannelBreakerConfig{FailureThreshold: 99, Cooldown: 30 * time.Second, WindowSeconds: 60, MinimumSamples: 4, ErrorRatePercent: 50, TimeoutRatePercent: 75}
	if ShouldOpenChannelBreaker(ChannelHealthWindow{Requests: 3, Errors: 3}, cfg) {
		t.Fatal("opened with insufficient samples")
	}
	if !ShouldOpenChannelBreaker(ChannelHealthWindow{Requests: 4, Errors: 2}, cfg) {
		t.Fatal("did not open at error threshold")
	}
	if !ShouldOpenChannelBreaker(ChannelHealthWindow{Requests: 4, Timeouts: 3}, cfg) {
		t.Fatal("did not open at timeout threshold")
	}
}

func TestWindowErrorRateOpensBreakerRegardlessOfOrder(t *testing.T) {
	cfg := ChannelBreakerConfig{FailureThreshold: 99, Cooldown: 30 * time.Second, WindowSeconds: 60, MinimumSamples: 10, ErrorRatePercent: 50, TimeoutRatePercent: 50}
	health := NewChannelHealth(1)
	// Interleaved successes reset the consecutive counter, but the window error
	// rate stays above threshold so the breaker must still open.
	for i := 0; i < 5; i++ {
		health = ApplyChannelSuccess(health, breakerTestNow)
		health = ApplyChannelFailure(health, FailureUpstream5xx, ChannelHealthWindow{Requests: 10, Errors: 6}, breakerTestNow, cfg)
	}
	if health.State != HealthOpen {
		t.Fatalf("state = %s, want open via window rate", health.State)
	}
}

func TestConsecutiveFailureFallbackBelowMinimumSamples(t *testing.T) {
	cfg := ChannelBreakerConfig{FailureThreshold: 3, Cooldown: 30 * time.Second, WindowSeconds: 60, MinimumSamples: 100, ErrorRatePercent: 50, TimeoutRatePercent: 50}
	health := NewChannelHealth(1)
	for i := 0; i < 3; i++ {
		health = ApplyChannelFailure(health, FailureUpstream5xx, ChannelHealthWindow{Requests: 2, Errors: 2}, breakerTestNow, cfg)
	}
	if health.State != HealthOpen {
		t.Fatalf("state = %s, want open via consecutive fallback", health.State)
	}
}

func TestResolveChannelBreakerConfigOverrides(t *testing.T) {
	base := DefaultChannelBreakerConfig()
	override := ChannelBreakerConfig{WindowSeconds: 120, ErrorRatePercent: 20}
	got := ResolveChannelBreakerConfig(base, &override)
	if got.WindowSeconds != 120 || got.ErrorRatePercent != 20 {
		t.Fatalf("override not applied: %+v", got)
	}
	if got.FailureThreshold != base.FailureThreshold || got.Cooldown != base.Cooldown || got.MinimumSamples != base.MinimumSamples || got.TimeoutRatePercent != base.TimeoutRatePercent {
		t.Fatalf("unset fields must inherit defaults: %+v", got)
	}
}

func TestOnlyUpstreamFailuresCountAsChannelFailures(t *testing.T) {
	if FailureReason("").CountsAsChannelFailure() {
		t.Fatal("zero reason must not count as a channel failure")
	}
	for _, reason := range []FailureReason{
		FailureUpstreamUnreachable, FailureUpstreamTimeout, FailureUpstreamProtocol,
		FailureUpstream401, FailureUpstream402, FailureUpstream403,
		FailureUpstream429, FailureUpstream5xx,
	} {
		if !reason.CountsAsChannelFailure() {
			t.Fatalf("%s must count as a channel failure", reason)
		}
	}
}
