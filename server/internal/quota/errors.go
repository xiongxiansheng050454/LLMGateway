package quota

import apperrors "LLMGateway/server/internal/errors"

var (
	ErrInvalid       = apperrors.ErrInvalid
	ErrQuotaExceeded = apperrors.ErrQuotaExceeded
)
