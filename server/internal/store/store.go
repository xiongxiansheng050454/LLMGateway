package store

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrInvalid        = errors.New("invalid")
	ErrNotImplemented = errors.New("not implemented")
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
}
