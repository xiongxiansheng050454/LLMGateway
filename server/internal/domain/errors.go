package domain

import "errors"

// Shared domain errors. The store package re-exports them so the persistence
// port keeps its error vocabulary while validation rules live in domain.
var (
	ErrNotFound       = errors.New("not found")
	ErrInvalid        = errors.New("invalid")
	ErrNotImplemented = errors.New("not implemented")
)
