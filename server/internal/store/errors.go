package store

import "LLMGateway/server/internal/errors"

var (
	ErrNotFound       = errors.ErrNotFound
	ErrInvalid        = errors.ErrInvalid
	ErrNotImplemented = errors.ErrNotImplemented
	ErrQuotaExceeded  = errors.ErrQuotaExceeded
)
