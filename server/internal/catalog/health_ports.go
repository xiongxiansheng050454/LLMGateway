package catalog

import (
	"context"
	"time"
)

// ChannelHealthStore persists per-channel circuit breaker state. Reading health
// lazily evaluates the open -> half-open transition; the transition is only
// persisted on the next recorded success/failure. The breaker state machine
// is catalog-owned pure business logic.
// HealthPort persists circuit breaker state for upstream channels.
type HealthPort interface {
	GetChannelHealth(channelID int) (ChannelHealth, error)
	RecordChannelSuccess(channelID int) (ChannelHealth, error)
	RecordChannelFailure(channelID int, reason FailureReason) (ChannelHealth, error)
	ResetChannelHealth(channelID int) error
	ListChannelHealth() (ListResponse[ChannelHealthDTO], error)
	RecordChannelAttempt(context.Context, int, bool, FailureReason) (ChannelHealth, error)
	AcquireChannelProbe(context.Context, int, time.Duration) (bool, error)
}
