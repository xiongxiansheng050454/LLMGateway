package store

import "LLMGateway/server/internal/domain"

// ChannelHealthStore persists per-channel circuit breaker state. Reading health
// lazily evaluates the open -> half-open transition; the transition is only
// persisted on the next recorded success/failure. The breaker state machine
// itself lives in domain (pure rules).
type ChannelHealthStore interface {
	GetChannelHealth(channelID int) (domain.ChannelHealth, error)
	RecordChannelSuccess(channelID int) (domain.ChannelHealth, error)
	RecordChannelFailure(channelID int, reason domain.FailureReason) (domain.ChannelHealth, error)
	ResetChannelHealth(channelID int) error
	ListChannelHealth() (domain.ListResponse[domain.ChannelHealthDTO], error)
}
