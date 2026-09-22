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
	ListChannelHealthRows() ([]ChannelHealth, error)
	// GetChannelHealthRow returns the stored row and whether it exists.
	GetChannelHealthRow(channelID int) (ChannelHealth, bool, error)
	AcquireChannelProbe(ctx context.Context, channelID int, lease time.Duration) (bool, error)
}

// Tx is the transaction-scoped persistence surface for catalog writes. It
// exposes only primitives; Server owns the state transitions and balance math.
type Tx interface {
	EnsureChannelHealth(channelID int) error
	GetChannelHealthForUpdate(channelID int) (ChannelHealth, error)
	UpdateChannelHealth(health ChannelHealth) (bool, error)
	DeleteChannelHealth(channelID int) error

	LockChannel(channelID int) error
	GetChannelBalanceText(channelID int) (string, error)
	UpdateChannelBalance(channelID int, balance string) (bool, error)
}

// TxManager runs fn inside a single database transaction.
type TxManager interface {
	InTx(ctx context.Context, fn func(Tx) error) error
}
