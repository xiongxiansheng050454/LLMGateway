package quota

import apperrors "LLMGateway/server/internal/errors"

var (
	ErrNotFound      = apperrors.ErrNotFound
	ErrInvalid       = apperrors.ErrInvalid
	ErrQuotaExceeded = apperrors.ErrQuotaExceeded
)
