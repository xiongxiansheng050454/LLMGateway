package postgres

import (
	"context"
	"errors"
	"testing"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/store"
)

func TestPGUsageLogsAndStats(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	if _, err := st.CreateChannel(domain.ChannelInput{Name: "OpenAI", BaseURL: "https://api.test", APIKey: "sk", Status: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateUser(domain.UserInput{Nickname: "A"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateUser(domain.UserInput{Nickname: "B"}); err != nil {
		t.Fatal(err)
	}

	first, err := st.InsertUsageLog(domain.UsageLogInput{RequestID: "req-1", UserID: intp(1), ChannelID: intp(1), Model: "gpt", Status: "success", TotalTokens: 100, TotalCost: "0.001000", UnitPriceInputPer1M: "0.10000000", UnitPriceOutputPer1M: "0.20000000"})
	if err != nil {
		t.Fatalf("InsertUsageLog: %v", err)
	}
	second, err := st.InsertUsageLog(domain.UsageLogInput{RequestID: "req-2", UserID: intp(1), ChannelID: intp(1), Model: "gpt-4o", Status: "error", TotalTokens: 200, TotalCost: "0.002000"})
	if err != nil {
		t.Fatal(err)
	}
	third, err := st.InsertUsageLog(domain.UsageLogInput{RequestID: "req-3", UserID: intp(2), ChannelID: intp(1), Model: "gpt", Status: "success", TotalTokens: 300, TotalCost: "0.003000"})
	if err != nil {
		t.Fatal(err)
	}

	// Control timestamps for deterministic UTC day grouping.
	if _, err := st.pool.Exec(ctx, "UPDATE usage_logs SET created_at = '2026-09-16T10:00:00Z' WHERE id = $1", first); err != nil {
		t.Fatal(err)
	}
	if _, err := st.pool.Exec(ctx, "UPDATE usage_logs SET created_at = '2026-09-16T23:30:00Z' WHERE id = $1", second); err != nil {
		t.Fatal(err)
	}
	if _, err := st.pool.Exec(ctx, "UPDATE usage_logs SET created_at = '2026-09-17T10:00:00Z' WHERE id = $1", third); err != nil {
		t.Fatal(err)
	}

	// List + filter.
	all, err := st.ListUsageLogs(domain.UsageLogFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListUsageLogs: %v", err)
	}
	if all.Total != 3 {
		t.Fatalf("total = %d, want 3", all.Total)
	}
	row := all.List[0].(map[string]any)
	if row["channel_name"] != "OpenAI" || row["total_cost"] == "" {
		t.Fatalf("unexpected row: %+v", row)
	}

	filtered, _ := st.ListUsageLogs(domain.UsageLogFilter{Status: "error", Model: "gpt-4o", Page: 1, PageSize: 20})
	if filtered.Total != 1 {
		t.Fatalf("filtered total = %d, want 1", filtered.Total)
	}
	byUser, _ := st.ListUsageLogs(domain.UsageLogFilter{UserID: intp(1), Page: 1, PageSize: 20})
	if byUser.Total != 2 {
		t.Fatalf("user filter total = %d, want 2", byUser.Total)
	}
	byRange, _ := st.ListUsageLogs(domain.UsageLogFilter{StartTime: "2026-09-16T00:00:00Z", EndTime: "2026-09-16T23:59:59Z", Page: 1, PageSize: 20})
	if byRange.Total != 2 {
		t.Fatalf("range filter total = %d, want 2", byRange.Total)
	}

	if _, err := st.GetUsageLog(first); err != nil {
		t.Fatalf("GetUsageLog: %v", err)
	}
	if _, err := st.GetUsageLog(404); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing log err = %v, want ErrNotFound", err)
	}

	overview, err := st.StatsOverview("2026-09-16T00:00:00Z", "2026-09-16T23:59:59Z")
	if err != nil {
		t.Fatalf("StatsOverview: %v", err)
	}
	if overview["request_count"] != int64(2) || overview["success_count"] != int64(1) || overview["error_count"] != int64(1) {
		t.Fatalf("unexpected overview: %+v", overview)
	}
	if overview["total_cost"] != "0.003000" || overview["active_user_count"] != int64(1) {
		t.Fatalf("unexpected overview aggregates: %+v", overview)
	}

	daily, err := st.StatsDaily("2026-09-16", "2026-09-17", 1, 100)
	if err != nil {
		t.Fatalf("StatsDaily: %v", err)
	}
	if daily.Total != 2 {
		t.Fatalf("daily total = %d, want 2", daily.Total)
	}
	day := daily.List[0].(map[string]any)
	if day["stat_date"] != "2026-09-16" || day["request_count"] != int64(2) || day["total_cost"] != "0.003000" {
		t.Fatalf("unexpected day: %+v", day)
	}

	channels, err := st.StatsChannels("2026-09-16T00:00:00Z", "2026-09-16T23:59:59Z")
	if err != nil {
		t.Fatalf("StatsChannels: %v", err)
	}
	if len(channels.List) != 1 {
		t.Fatalf("channels len = %d, want 1", len(channels.List))
	}
	channel := channels.List[0].(map[string]any)
	if channel["channel_name"] != "OpenAI" || channel["request_count"] != int64(2) {
		t.Fatalf("unexpected channel stat: %+v", channel)
	}
}

func TestPGUsageInvalidTimeParams(t *testing.T) {
	st := testStore(t)

	if _, err := st.ListUsageLogs(domain.UsageLogFilter{StartTime: "abc", Page: 1, PageSize: 20}); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid start_time err = %v, want ErrInvalid", err)
	}
	if _, err := st.StatsOverview("abc", ""); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid overview start err = %v, want ErrInvalid", err)
	}
	if _, err := st.StatsChannels("", "abc"); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid channels end err = %v, want ErrInvalid", err)
	}
	if _, err := st.StatsDaily("abc", "2026-01-01", 1, 100); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("invalid date_from err = %v, want ErrInvalid", err)
	}
}

func TestPGStatsEmptyReturnsZeroValues(t *testing.T) {
	st := testStore(t)

	overview, err := st.StatsOverview("", "")
	if err != nil {
		t.Fatalf("StatsOverview: %v", err)
	}
	if overview["request_count"] != int64(0) || overview["total_cost"] != "0.000000" {
		t.Fatalf("unexpected empty overview: %+v", overview)
	}

	daily, err := st.StatsDaily("1970-01-01", "9999-12-31", 1, 100)
	if err != nil {
		t.Fatalf("StatsDaily: %v", err)
	}
	if daily.Total != 0 || len(daily.List) != 0 {
		t.Fatalf("unexpected empty daily: %+v", daily)
	}

	channels, err := st.StatsChannels("", "")
	if err != nil {
		t.Fatalf("StatsChannels: %v", err)
	}
	if len(channels.List) != 0 {
		t.Fatalf("unexpected empty channels: %+v", channels)
	}
}
