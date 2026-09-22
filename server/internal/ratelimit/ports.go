package ratelimit

// Port is the rate-limit persistence primitive surface. It exposes only CRUD
// and query primitives: normalization and validation live on Server.
type Port interface {
	ListRateLimits(enabled *bool, page, pageSize int) (ListResponse[RateLimitRuleDTO], error)
	GetRateLimit(id int) (RateLimitRule, error)
	InsertRateLimit(rule RateLimitRule) (int, error)
	UpdateRateLimitRecord(id int, rule RateLimitRule) (bool, error)
	DeleteRateLimit(id int) (bool, error)

	InsertRateLimitReservation(in RateLimitReservationInput) (int64, error)
	FinalizeRateLimitReservation(id int64) (bool, error)
	ReleaseRateLimitReservation(id int64) (bool, error)
	ReapRateLimitReservations(limit int) (int, error)
	CountActiveRateLimitReservations(userID int, apiKeyID *int, model string, channelID *int) (int64, error)
}
