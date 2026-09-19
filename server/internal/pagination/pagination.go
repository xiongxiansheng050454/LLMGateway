// Package pagination provides protocol-neutral list result shapes for business
// ports. It deliberately has no HTTP or persistence dependencies.
package pagination

// List is the shared shape for paginated business results.
type List[T any] struct {
	List  []T `json:"list"`
	Total int `json:"total"`
}
