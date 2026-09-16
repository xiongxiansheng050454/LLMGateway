package store

import (
	"fmt"
	"time"

	"LLMGateway/internal/domain"
)

// ValidateTimeRange checks optional RFC3339 start/end bounds. Invalid values
// return ErrInvalid so memory and PostgreSQL behave identically.
func ValidateTimeRange(startTime, endTime string) error {
	if startTime != "" {
		if _, err := time.Parse(time.RFC3339, startTime); err != nil {
			return fmt.Errorf("%w: invalid start_time", ErrInvalid)
		}
	}
	if endTime != "" {
		if _, err := time.Parse(time.RFC3339, endTime); err != nil {
			return fmt.Errorf("%w: invalid end_time", ErrInvalid)
		}
	}
	return nil
}

// ValidateDateRange checks optional YYYY-MM-DD bounds.
func ValidateDateRange(dateFrom, dateTo string) error {
	if dateFrom != "" {
		if _, err := time.Parse("2006-01-02", dateFrom); err != nil {
			return fmt.Errorf("%w: invalid date_from", ErrInvalid)
		}
	}
	if dateTo != "" {
		if _, err := time.Parse("2006-01-02", dateTo); err != nil {
			return fmt.Errorf("%w: invalid date_to", ErrInvalid)
		}
	}
	return nil
}

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
}
