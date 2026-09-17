package store

import "LLMGateway/server/internal/domain"

// UsageStore records and queries request usage logs, and aggregates statistics
// from them. Statistics are computed from usage_logs in real time; the
// daily_usage_stats table was dropped (migration 000004) because it was unused
// and its composite primary key could not represent global per-day rows.
type UsageStore interface {
	ListUsageLogs(domain.UsageLogFilter) (domain.ListResponse, error)
	GetUsageLog(int) (map[string]any, error)
	InsertUsageLog(domain.UsageLogInput) (int, error)
	StatsOverview(startTime, endTime string) (map[string]any, error)
	StatsDaily(dateFrom, dateTo string, page, pageSize int) (domain.ListResponse, error)
	StatsChannels(startTime, endTime string) (domain.ListResponse, error)

	// CountRequestsSince counts request attempts for a user (optionally scoped
	// to a key) since an RFC3339 timestamp. It counts all attempts, including
	// failures, for window-based rate limiting.
	CountRequestsSince(userID int, apiKeyID *int, since string) (int, error)
}
