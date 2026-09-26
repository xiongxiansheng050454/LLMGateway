package ratelimit

import "context"

// Port is the rate-limit persistence primitive surface. It exposes only CRUD
// and query primitives: normalization and validation live on Server.
type Port interface {
	ListRateLimits(ctx context.Context, enabled *bool, page, pageSize int) (ListResponse[RateLimitRuleDTO], error)
	GetRateLimit(ctx context.Context, id int) (RateLimitRule, error)
	InsertRateLimit(ctx context.Context, rule RateLimitRule) (int, error)
	UpdateRateLimitRecord(ctx context.Context, id int, rule RateLimitRule) (bool, error)
	DeleteRateLimit(ctx context.Context, id int) (bool, error)

	InsertRateLimitReservation(ctx context.Context, in RateLimitReservationInput) (int64, error)
	FinalizeRateLimitReservation(ctx context.Context, id int64) (bool, error)
	ReleaseRateLimitReservation(ctx context.Context, id int64) (bool, error)
	ReapRateLimitReservations(ctx context.Context, limit int) (int, error)
	CountActiveRateLimitReservations(ctx context.Context, userID int, apiKeyID *int, model string, channelID *int) (int64, error)
}
