package catalog

import (
	"context"
	"time"
)

// HealthPort persists per-channel circuit breaker state. Reading health lazily
// evaluates the open -> half-open transition; the transition is only persisted
// on the next recorded success/failure. The breaker state machine is
// catalog-owned pure business logic.
type HealthPort interface {
	ListChannelHealthRows(ctx context.Context) ([]ChannelHealth, error)
	// GetChannelHealthRow returns the stored row and whether it exists.
	GetChannelHealthRow(ctx context.Context, channelID int) (ChannelHealth, bool, error)
	AcquireChannelProbe(ctx context.Context, channelID int, lease time.Duration) (bool, error)
}
