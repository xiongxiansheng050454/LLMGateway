package store

import (
	"time"

	"LLMGateway/server/internal/domain"
)

// ChannelHealthStore persists per-channel circuit breaker state. Reading health
// lazily evaluates the open -> half-open transition; the transition is only
// persisted on the next recorded success/failure.
type ChannelHealthStore interface {
	GetChannelHealth(channelID int) (domain.ChannelHealth, error)
	RecordChannelSuccess(channelID int) (domain.ChannelHealth, error)
	RecordChannelFailure(channelID int, reason string) (domain.ChannelHealth, error)
	ResetChannelHealth(channelID int) error
	ListChannelHealth() (domain.ListResponse, error)
}

// ChannelBreakerConfig holds the circuit breaker thresholds shared by the
// memory and PostgreSQL implementations.
type ChannelBreakerConfig struct {
	FailureThreshold int
	Cooldown         time.Duration
}

func DefaultChannelBreakerConfig() ChannelBreakerConfig {
	return ChannelBreakerConfig{FailureThreshold: 5, Cooldown: 30 * time.Second}
}

// IsDeterministicChannelFailure reports whether a failure reason should trip the
// breaker immediately instead of counting toward the threshold (auth/quota
// problems that will not recover on retry).
func IsDeterministicChannelFailure(reason string) bool {
	switch reason {
	case "upstream_401", "upstream_403", "upstream_402", "invalid_channel_key":
		return true
	default:
		return false
	}
}

// NewChannelHealth returns the default closed state for a channel.
func NewChannelHealth(channelID int) domain.ChannelHealth {
	return domain.ChannelHealth{ChannelID: channelID, State: domain.HealthClosed}
}

// EvaluateChannelHealth lazily moves an open channel to half-open once the
// cooldown has elapsed. It is a pure function; callers decide whether to persist.
func EvaluateChannelHealth(current domain.ChannelHealth, now time.Time, cfg ChannelBreakerConfig) domain.ChannelHealth {
	if current.State != domain.HealthOpen || current.OpenedAt == nil {
		return current
	}
	openedAt, err := time.Parse(time.RFC3339, *current.OpenedAt)
	if err != nil {
		return current
	}
	if now.Before(openedAt.Add(cfg.Cooldown)) {
		return current
	}
	current.State = domain.HealthHalfOpen
	current.ConsecutiveFailures = 0
	current.OpenedAt = nil
	return current
}

// ApplyChannelSuccess records a successful attempt; any state returns to closed.
func ApplyChannelSuccess(current domain.ChannelHealth, now time.Time) domain.ChannelHealth {
	current.SuccessCount++
	current.ConsecutiveFailures = 0
	current.State = domain.HealthClosed
	current.OpenedAt = nil
	current.UpdatedAt = now.UTC().Format(time.RFC3339)
	return current
}

// ApplyChannelFailure records a failed attempt and may trip the breaker.
func ApplyChannelFailure(current domain.ChannelHealth, reason string, now time.Time, cfg ChannelBreakerConfig) domain.ChannelHealth {
	current.FailureCount++
	current.ConsecutiveFailures++
	current.UpdatedAt = now.UTC().Format(time.RFC3339)

	shouldOpen := IsDeterministicChannelFailure(reason) ||
		current.State == domain.HealthHalfOpen ||
		current.ConsecutiveFailures >= cfg.FailureThreshold
	if shouldOpen {
		current.State = domain.HealthOpen
		opened := now.UTC().Format(time.RFC3339)
		current.OpenedAt = &opened
	}
	return current
}
