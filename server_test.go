package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	NewHandler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var body map[string]string
	decodeJSON(t, res, &body)
	if body["status"] != "ok" {
		t.Fatalf("status field = %q, want ok", body["status"])
	}
}

func TestDashboardStartupEndpoints(t *testing.T) {
	tests := []struct {
		path         string
		wantFields   []string
		wantListData bool
	}{
		{"/admin/stats/overview?start_time=2026-09-16T10:00:00Z&end_time=2026-09-16T11:00:00Z", []string{"request_count", "success_count", "error_count", "total_tokens", "total_cost", "active_user_count"}, false},
		{"/admin/stats/daily?date_from=2026-09-10&date_to=2026-09-16&page=1&page_size=100", nil, true},
		{"/admin/channels?page=1&page_size=100", nil, true},
		{"/admin/stats/channels?start_time=2026-09-16T10:00:00Z&end_time=2026-09-16T11:00:00Z", []string{"list"}, false},
		{"/admin/usage-logs?page=1&page_size=20", nil, true},
		{"/admin/users?page=1&page_size=100", nil, true},
		{"/admin/rate-limits?page=1&page_size=100&enabled=true", nil, true},
		{"/admin/models?status=1", nil, true},
	}

	handler := NewHandler()
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			res := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			handler.ServeHTTP(res, req)

			if res.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusOK, res.Body.String())
			}

			var body adminResponse
			decodeJSON(t, res, &body)
			if body.Code != 0 || body.Message != "ok" {
				t.Fatalf("response = %+v, want code=0 message=ok", body)
			}

			data, ok := body.Data.(map[string]any)
			if !ok {
				t.Fatalf("data type = %T, want object", body.Data)
			}

			if tt.wantListData {
				assertListResponse(t, data)
			}
			for _, field := range tt.wantFields {
				if _, ok := data[field]; !ok {
					t.Fatalf("missing data.%s in %+v", field, data)
				}
			}
		})
	}
}

func TestPaginationParsing(t *testing.T) {
	page, pageSize := parsePagination(httptest.NewRequest(http.MethodGet, "/?page=2&page_size=50", nil))
	if page != 2 || pageSize != 50 {
		t.Fatalf("parsePagination valid = %d,%d; want 2,50", page, pageSize)
	}

	page, pageSize = parsePagination(httptest.NewRequest(http.MethodGet, "/?page=-1&page_size=abc", nil))
	if page != 1 || pageSize != 20 {
		t.Fatalf("parsePagination fallback = %d,%d; want 1,20", page, pageSize)
	}
}

func TestStaticDashboardServed(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	NewHandler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if res.Body.Len() == 0 {
		t.Fatal("empty dashboard response")
	}
}

func assertListResponse(t *testing.T, data map[string]any) {
	t.Helper()
	if _, ok := data["list"].([]any); !ok {
		t.Fatalf("data.list type = %T, want array", data["list"])
	}
	if _, ok := data["total"].(float64); !ok {
		t.Fatalf("data.total type = %T, want number", data["total"])
	}
}

func decodeJSON(t *testing.T, res *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(res.Body.Bytes(), v); err != nil {
		t.Fatalf("decode JSON: %v; body=%s", err, res.Body.String())
	}
}
