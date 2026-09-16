package store

import "LLMGateway/internal/domain"

// UsageStore records and queries request usage logs, and aggregates statistics
// from them. Statistics are computed from usage_logs in real time; the
// daily_usage_stats table is currently kept but unused (its composite primary
// key cannot represent global per-day rows).
type UsageStore interface {
	ListUsageLogs(domain.UsageLogFilter) (domain.ListResponse, error)
	GetUsageLog(int) (map[string]any, error)
	InsertUsageLog(domain.UsageLogInput) (int, error)
	StatsOverview(startTime, endTime string) (map[string]any, error)
	StatsDaily(dateFrom, dateTo string, page, pageSize int) (domain.ListResponse, error)
	StatsChannels(startTime, endTime string) (domain.ListResponse, error)
}
