package domain

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
