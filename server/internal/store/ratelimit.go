package store

import "LLMGateway/server/internal/domain"

// RateLimitStore manages rate limit rules. This issue only stores rules; it does
// not enforce them at request time.
type RateLimitStore interface {
	ListRateLimits(enabled *bool, page, pageSize int) (domain.ListResponse[domain.RateLimitRuleDTO], error)
	CreateRateLimit(domain.RateLimitInput) (domain.RateLimitRuleDTO, error)
	UpdateRateLimit(int, domain.RateLimitInput) (domain.RateLimitRuleDTO, error)
	DeleteRateLimit(int) error
}
