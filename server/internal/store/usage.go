package store

import "LLMGateway/server/internal/domain"

// UsageStore records and queries request usage logs, and aggregates statistics
// from them. Statistics are computed from usage_logs in real time; the
// daily_usage_stats table was dropped (migration 000004) because it was unused
// and its composite primary key could not represent global per-day rows.
type UsageStore interface {
	ListUsageLogs(domain.UsageLogFilter) (domain.ListResponse[domain.UsageLogDTO], error)
	GetUsageLog(int) (domain.UsageLogDTO, error)
	InsertUsageLog(domain.UsageLogInput) (int, error)
	SettleChatCompletion(domain.ChatSettlementInput) (int, error)
	StatsOverview(startTime, endTime string) (domain.StatsOverviewDTO, error)
	StatsDaily(dateFrom, dateTo string, page, pageSize int) (domain.ListResponse[domain.StatsDailyDTO], error)
	StatsChannels(startTime, endTime string) (domain.ListResponse[domain.StatsChannelDTO], error)
	StatsTTFT(domain.TTFTStatsFilter) (domain.TTFTStatsDTO, error)

	// CountRequestsSince counts request attempts matching the filter since an
	// RFC3339 timestamp. It counts all attempts, including failures, for
	// window-based rate limiting.
	CountRequestsSince(domain.UsageCountFilter) (int, error)
}
