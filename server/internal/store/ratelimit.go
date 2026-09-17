package store

import "LLMGateway/server/internal/domain"

// RateLimitStore manages rate limit rules. This issue only stores rules; it does
// not enforce them at request time.
type RateLimitStore interface {
	ListRateLimits(enabled *bool, page, pageSize int) (domain.ListResponse, error)
	CreateRateLimit(domain.RateLimitInput) (map[string]any, error)
	UpdateRateLimit(int, domain.RateLimitInput) (map[string]any, error)
	DeleteRateLimit(int) error
}
