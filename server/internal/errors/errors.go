package errors

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrInvalid        = errors.New("invalid")
	ErrNotImplemented = errors.New("not implemented")
	ErrQuotaExceeded  = errors.New("quota exceeded")
)
