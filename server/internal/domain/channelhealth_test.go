package domain

import (
	"testing"
	"time"
)

var breakerTestNow = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)

func TestApplyChannelFailureOpensAtThreshold(t *testing.T) {
	cfg := ChannelBreakerConfig{FailureThreshold: 3, Cooldown: 30 * time.Second}
	health := NewChannelHealth(1)

	for i := 1; i <= 2; i++ {
		health = ApplyChannelFailure(health, FailureUpstream5xx, breakerTestNow, cfg)
		if health.State != HealthClosed {
			t.Fatalf("after %d failures state = %s, want closed", i, health.State)
		}
	}
	health = ApplyChannelFailure(health, FailureUpstream5xx, breakerTestNow, cfg)
	if health.State != HealthOpen {
		t.Fatalf("state = %s, want open", health.State)
	}
	if health.OpenedAt == nil {
		t.Fatal("opened_at not set")
	}
	if health.ConsecutiveFailures != 3 || health.FailureCount != 3 {
		t.Fatalf("counters = %d/%d, want 3/3", health.ConsecutiveFailures, health.FailureCount)
	}
}

func TestDeterministicFailureOpensImmediately(t *testing.T) {
	cfg := ChannelBreakerConfig{FailureThreshold: 5, Cooldown: 30 * time.Second}
	health := ApplyChannelFailure(NewChannelHealth(1), FailureUpstream401, breakerTestNow, cfg)
	if health.State != HealthOpen {
		t.Fatalf("state = %s, want open for deterministic failure", health.State)
	}
}

func TestEvaluateTransitionsToHalfOpenAfterCooldown(t *testing.T) {
	cfg := ChannelBreakerConfig{FailureThreshold: 5, Cooldown: 30 * time.Second}
	opened := breakerTestNow.Format(time.RFC3339)
	health := ChannelHealth{ChannelID: 1, State: HealthOpen, OpenedAt: &opened, ConsecutiveFailures: 5}

	if got := EvaluateChannelHealth(health, breakerTestNow.Add(29*time.Second), cfg); got.State != HealthOpen {
		t.Fatalf("before cooldown state = %s, want open", got.State)
	}
	got := EvaluateChannelHealth(health, breakerTestNow.Add(30*time.Second), cfg)
	if got.State != HealthHalfOpen {
		t.Fatalf("after cooldown state = %s, want half-open", got.State)
	}
	if got.ConsecutiveFailures != 0 || got.OpenedAt != nil {
		t.Fatalf("half-open should reset consecutive failures and opened_at: %+v", got)
	}
}

func TestHalfOpenSuccessClosesAndFailureReopens(t *testing.T) {
	cfg := ChannelBreakerConfig{FailureThreshold: 5, Cooldown: 30 * time.Second}
	halfOpen := ChannelHealth{ChannelID: 1, State: HealthHalfOpen, FailureCount: 5}

	closed := ApplyChannelSuccess(halfOpen, breakerTestNow)
	if closed.State != HealthClosed || closed.ConsecutiveFailures != 0 || closed.OpenedAt != nil {
		t.Fatalf("unexpected closed state: %+v", closed)
	}

	reopened := ApplyChannelFailure(halfOpen, FailureUpstream5xx, breakerTestNow, cfg)
	if reopened.State != HealthOpen {
		t.Fatalf("half-open failure state = %s, want open", reopened.State)
	}
}
