package domain

import (
	"fmt"
	"time"
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

// ValidateSince requires a non-empty RFC3339 timestamp. Used by window
// counting so memory and PostgreSQL cannot diverge on an empty value.
func ValidateSince(since string) error {
	if since == "" {
		return fmt.Errorf("%w: since is required", ErrInvalid)
	}
	return ValidateTimeRange(since, "")
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

type UsageLog struct {
	ID                   int
	RequestID            string
	UserID               *int
	APIKeyID             *int
	ChannelID            *int
	ChannelName          string
	Model                string
	UpstreamModel        string
	InputTokens          int
	OutputTokens         int
	CachedInputTokens    int
	TotalTokens          int
	UnitPriceInputPer1M  string
	UnitPriceOutputPer1M string
	TotalCost            string
	DurationMs           int
	TTFTMs               *int
	Status               string
	ErrorCode            string
	ClientIP             string
	CreatedAt            string
}

type UsageLogFilter struct {
	UserID    *int
	ChannelID *int
	Model     string
	Status    string
	StartTime string
	EndTime   string
	Page      int
	PageSize  int
}

// UsageLogInput is the write model reused by the downstream proxy (#6).
type UsageLogInput struct {
	RequestID            string
	UserID               *int
	APIKeyID             *int
	ChannelID            *int
	Model                string
	UpstreamModel        string
	InputTokens          int
	OutputTokens         int
	CachedInputTokens    int
	TotalTokens          int
	UnitPriceInputPer1M  string
	UnitPriceOutputPer1M string
	TotalCost            string
	DurationMs           int
	TTFTMs               *int
	Status               string
	ErrorCode            string
	ClientIP             string
}
