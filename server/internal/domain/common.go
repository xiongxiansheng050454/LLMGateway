package domain

// ListResponse is the shared shape for paginated list endpoints.
type ListResponse[T any] struct {
	List  []T `json:"list"`
	Total int `json:"total"`
}
