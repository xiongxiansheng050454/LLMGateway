package store

import "LLMGateway/server/internal/domain"

// The port re-exports the shared domain errors so existing callers keep using
// store.ErrNotFound / store.ErrInvalid / store.ErrNotImplemented.
var (
	ErrNotFound       = domain.ErrNotFound
	ErrInvalid        = domain.ErrInvalid
	ErrNotImplemented = domain.ErrNotImplemented
	ErrQuotaExceeded  = domain.ErrQuotaExceeded
)

// Store is the persistence port used by the HTTP handler layer.
//
// It composes per-domain interfaces so each domain can grow its own file
// without turning Store into a god interface. Implementations must not expose
// sensitive values (upstream channel api_key, gateway key plaintext) through
// response DTOs.
type Store interface {
	ChannelStore
	UserStore
	RateLimitStore
	UsageStore
	ChannelHealthStore
	QuotaStore
}
