package domain

// ListResponse is the shared shape for paginated list endpoints.
type ListResponse struct {
	List  []any `json:"list"`
	Total int   `json:"total"`
}
