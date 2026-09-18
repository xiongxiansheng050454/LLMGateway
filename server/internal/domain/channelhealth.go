package domain

import "time"

// HealthState is the circuit breaker state of an upstream channel.
type HealthState string

const (
	HealthClosed   HealthState = "closed"
	HealthOpen     HealthState = "open"
	HealthHalfOpen HealthState = "half-open"
)

// ChannelHealth is the persisted circuit breaker state for a channel. A missing
// row means "closed".
type ChannelHealth struct {
	ChannelID           int
	State               HealthState
	ConsecutiveFailures int
	SuccessCount        int64
	FailureCount        int64
	OpenedAt            *string
	UpdatedAt           string
}

type ChannelHealthDTO struct {
	ChannelID           int     `json:"channel_id"`
	State               string  `json:"state"`
	ConsecutiveFailures int     `json:"consecutive_failures"`
	SuccessCount        int64   `json:"success_count"`
	FailureCount        int64   `json:"failure_count"`
	OpenedAt            *string `json:"opened_at"`
	UpdatedAt           string  `json:"updated_at"`
}

// ChannelBreakerConfig holds the circuit breaker thresholds shared by the
// persistence implementations and test fakes.
type ChannelBreakerConfig struct {
	FailureThreshold int
	Cooldown         time.Duration
}

func DefaultChannelBreakerConfig() ChannelBreakerConfig {
	return ChannelBreakerConfig{FailureThreshold: 5, Cooldown: 30 * time.Second}
}

// NewChannelHealth returns the default closed state for a channel.
func NewChannelHealth(channelID int) ChannelHealth {
	return ChannelHealth{ChannelID: channelID, State: HealthClosed}
}

// EvaluateChannelHealth lazily moves an open channel to half-open once the
// cooldown has elapsed. It is a pure function; callers decide whether to persist.
func EvaluateChannelHealth(current ChannelHealth, now time.Time, cfg ChannelBreakerConfig) ChannelHealth {
	if current.State != HealthOpen || current.OpenedAt == nil {
		return current
	}
	openedAt, err := time.Parse(time.RFC3339, *current.OpenedAt)
	if err != nil {
		return current
	}
	if now.Before(openedAt.Add(cfg.Cooldown)) {
		return current
	}
	current.State = HealthHalfOpen
	current.ConsecutiveFailures = 0
	current.OpenedAt = nil
	return current
}

// ApplyChannelSuccess records a successful attempt; any state returns to closed.
func ApplyChannelSuccess(current ChannelHealth, now time.Time) ChannelHealth {
	current.SuccessCount++
	current.ConsecutiveFailures = 0
	current.State = HealthClosed
	current.OpenedAt = nil
	current.UpdatedAt = now.UTC().Format(time.RFC3339)
	return current
}

// ApplyChannelFailure records a failed attempt and may trip the breaker.
func ApplyChannelFailure(current ChannelHealth, reason FailureReason, now time.Time, cfg ChannelBreakerConfig) ChannelHealth {
	current.FailureCount++
	current.ConsecutiveFailures++
	current.UpdatedAt = now.UTC().Format(time.RFC3339)

	shouldOpen := reason.IsDeterministic() ||
		current.State == HealthHalfOpen ||
		current.ConsecutiveFailures >= cfg.FailureThreshold
	if shouldOpen {
		current.State = HealthOpen
		opened := now.UTC().Format(time.RFC3339)
		current.OpenedAt = &opened
	}
	return current
}
