package httpcommon

import "LLMGateway/server/internal/pagination"

// ListResponse is the shared shape for paginated list endpoints.
type ListResponse[T any] = pagination.List[T]
