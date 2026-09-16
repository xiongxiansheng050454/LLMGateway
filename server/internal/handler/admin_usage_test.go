package handler

import (
	"net/http"
	"path/filepath"
	"testing"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/store/memory"
)

func intPtr(value int) *int { return &value }

func newUsageTestHandler(t *testing.T) (http.Handler, *memory.Store) {
	t.Helper()
	st := memory.New()
	return newTestRouter(filepath.Join("..", "..", "..", "dashboard"), st), st
}

func TestUsageInvalidTimeParamsReturnBadRequest(t *testing.T) {
	handler := newTestHandler()

	for _, path := range []string{
		"/admin/stats/overview?start_time=abc",
		"/admin/stats/channels?end_time=abc",
		"/admin/stats/daily?date_from=abc",
		"/admin/stats/daily?date_to=abc",
		"/admin/usage-logs?start_time=abc",
		"/admin/usage-logs?end_time=abc",
	} {
		res := adminRaw(t, handler, http.MethodGet, path, nil)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want 400; body=%s", path, res.Code, res.Body.String())
		}
	}
}

func TestUsageLogsAndStats(t *testing.T) {
	handler, st := newUsageTestHandler(t)

	// Empty data must still return code=0 with zero values.
	overview := adminDo(t, handler, http.MethodGet, "/admin/stats/overview", nil)
	if overview["data"].(map[string]any)["request_count"].(float64) != 0 {
		t.Fatalf("empty overview = %+v", overview["data"])
	}
	daily := adminDo(t, handler, http.MethodGet, "/admin/stats/daily", nil)
	if daily["data"].(map[string]any)["total"].(float64) != 0 {
		t.Fatalf("empty daily = %+v", daily["data"])
	}
	channels := adminDo(t, handler, http.MethodGet, "/admin/stats/channels", nil)
	if len(channels["data"].(map[string]any)["list"].([]any)) != 0 {
		t.Fatalf("empty channels = %+v", channels["data"])
	}

	if _, err := st.InsertUsageLog(domain.UsageLogInput{RequestID: "req-1", UserID: intPtr(1), ChannelID: intPtr(1), Model: "gpt", Status: "success", TotalTokens: 100, TotalCost: "0.001000"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertUsageLog(domain.UsageLogInput{RequestID: "req-2", UserID: intPtr(1), ChannelID: intPtr(1), Model: "gpt-4o", Status: "error", TotalTokens: 200, TotalCost: "0.002000"}); err != nil {
		t.Fatal(err)
	}

	logs := adminDo(t, handler, http.MethodGet, "/admin/usage-logs?page=1&page_size=20", nil)
	listData := logs["data"].(map[string]any)
	if listData["total"].(float64) != 2 {
		t.Fatalf("logs total = %v, want 2", listData["total"])
	}
	row := listData["list"].([]any)[0].(map[string]any)
	if row["request_id"] == "" || row["total_cost"] == "" {
		t.Fatalf("unexpected log row: %+v", row)
	}

	single := adminDo(t, handler, http.MethodGet, "/admin/usage-logs/1", nil)
	if single["data"].(map[string]any)["request_id"] != "req-1" {
		t.Fatalf("unexpected single log: %+v", single["data"])
	}
	missing := adminRaw(t, handler, http.MethodGet, "/admin/usage-logs/404", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing log status = %d, want 404", missing.Code)
	}

	filtered := adminDo(t, handler, http.MethodGet, "/admin/usage-logs?status=error&model=gpt-4o", nil)
	if filtered["data"].(map[string]any)["total"].(float64) != 1 {
		t.Fatalf("filtered total = %+v", filtered["data"])
	}

	overview = adminDo(t, handler, http.MethodGet, "/admin/stats/overview", nil)
	ov := overview["data"].(map[string]any)
	if ov["request_count"].(float64) != 2 || ov["success_count"].(float64) != 1 || ov["error_count"].(float64) != 1 {
		t.Fatalf("unexpected overview: %+v", ov)
	}
	if ov["total_tokens"].(float64) != 300 || ov["total_cost"] != "0.003000" || ov["active_user_count"].(float64) != 1 {
		t.Fatalf("unexpected overview aggregates: %+v", ov)
	}

	daily = adminDo(t, handler, http.MethodGet, "/admin/stats/daily", nil)
	dailyData := daily["data"].(map[string]any)
	if dailyData["total"].(float64) != 1 {
		t.Fatalf("daily total = %+v", dailyData)
	}
	day := dailyData["list"].([]any)[0].(map[string]any)
	if day["request_count"].(float64) != 2 || day["total_cost"] != "0.003000" {
		t.Fatalf("unexpected day: %+v", day)
	}

	channels = adminDo(t, handler, http.MethodGet, "/admin/stats/channels", nil)
	channelList := channels["data"].(map[string]any)["list"].([]any)
	if len(channelList) != 1 {
		t.Fatalf("channels len = %d, want 1", len(channelList))
	}
	channel := channelList[0].(map[string]any)
	if channel["channel_id"].(float64) != 1 || channel["request_count"].(float64) != 2 {
		t.Fatalf("unexpected channel stat: %+v", channel)
	}
}
