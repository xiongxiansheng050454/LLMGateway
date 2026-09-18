package store

import (
	"context"

	"LLMGateway/server/internal/domain"
)

// RateLimitStore manages rate limit rules. This issue only stores rules; it does
// not enforce them at request time.
type RateLimitStore interface {
	ListRateLimits(enabled *bool, page, pageSize int) (domain.ListResponse[domain.RateLimitRuleDTO], error)
	CreateRateLimit(domain.RateLimitInput) (domain.RateLimitRuleDTO, error)
	UpdateRateLimit(int, domain.RateLimitInput) (domain.RateLimitRuleDTO, error)
	DeleteRateLimit(int) error
	ReserveRateLimit(context.Context, domain.RateLimitReservationInput) (domain.RateLimitReservation, error)
	FinalizeRateLimit(context.Context, int64, int64) error
	ReleaseRateLimit(context.Context, int64) error
	ReapRateLimitReservations(context.Context, int) (int, error)
	CountActiveRateLimitReservations(context.Context, int, *int, string, *int) (int64, error)
}
