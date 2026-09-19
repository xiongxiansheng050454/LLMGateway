package ratelimit

import (
	"context"
)

// RateLimitStore manages rate limit rules. This issue only stores rules; it does
// not enforce them at request time.
type Port interface {
	ListRateLimits(enabled *bool, page, pageSize int) (ListResponse[RateLimitRuleDTO], error)
	CreateRateLimit(RateLimitInput) (RateLimitRuleDTO, error)
	UpdateRateLimit(int, RateLimitInput) (RateLimitRuleDTO, error)
	DeleteRateLimit(int) error
	ReserveRateLimit(context.Context, RateLimitReservationInput) (RateLimitReservation, error)
	FinalizeRateLimit(context.Context, int64, int64) error
	ReleaseRateLimit(context.Context, int64) error
	ReapRateLimitReservations(context.Context, int) (int, error)
	CountActiveRateLimitReservations(context.Context, int, *int, string, *int) (int64, error)
}
