package usage

// UsageStore records and queries request usage logs, and aggregates statistics
// from them. Statistics are computed from usage_logs in real time; there is no
// daily_usage_stats table because it was unused and its composite primary key
// could not represent global per-day rows.
type Port interface {
	ListUsageLogs(UsageLogFilter) (ListResponse[UsageLogDTO], error)
	GetUsageLog(int) (UsageLogDTO, error)
	InsertUsageLog(UsageLogInput) (int, error)
	StatsOverview(startTime, endTime string) (StatsOverviewDTO, error)
	StatsDaily(dateFrom, dateTo string, page, pageSize int) (ListResponse[StatsDailyDTO], error)
	StatsChannels(startTime, endTime string) (ListResponse[StatsChannelDTO], error)
	StatsTTFT(TTFTStatsFilter) (TTFTStatsDTO, error)
	AggregateUsage(UsageAggregateFilter) (ListResponse[UsageAggregateDTO], error)

	// CountRequestsSince counts request attempts matching the filter since an
	// RFC3339 timestamp. It counts all attempts, including failures, for
	// window-based rate limiting.
	CountRequestsSince(UsageCountFilter) (int, error)
	CountTokensSince(TokenCountFilter) (int64, error)
}
