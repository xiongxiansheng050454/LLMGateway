package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestHealthz(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	newTestHandler().ServeHTTP(res, req)

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

	handler := newTestHandler()
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

func TestAdminCORS(t *testing.T) {
	handler := newTestHandler()

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/stats/overview", nil)
	req.Header.Set("Origin", "http://example.test")
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "http://example.test" {
		t.Fatalf("GET Access-Control-Allow-Origin = %q", got)
	}

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodOptions, "/admin/stats/overview", nil)
	req.Header.Set("Origin", "http://example.test")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want %d", res.Code, http.StatusNoContent)
	}
	if got := res.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodGet) || !strings.Contains(got, http.MethodOptions) {
		t.Fatalf("OPTIONS Access-Control-Allow-Methods = %q", got)
	}
}

func TestPaginationParsing(t *testing.T) {
	page, pageSize := ParsePagination(httptest.NewRequest(http.MethodGet, "/?page=2&page_size=50", nil))
	if page != 2 || pageSize != 50 {
		t.Fatalf("ParsePagination valid = %d,%d; want 2,50", page, pageSize)
	}

	page, pageSize = ParsePagination(httptest.NewRequest(http.MethodGet, "/?page=-1&page_size=abc", nil))
	if page != 1 || pageSize != 20 {
		t.Fatalf("ParsePagination fallback = %d,%d; want 1,20", page, pageSize)
	}
}

func TestStaticDashboardServed(t *testing.T) {
	tests := []string{"/", "/dashboard/index.html", "/dashboard/js/data.js"}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			res := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			newTestHandler().ServeHTTP(res, req)

			if res.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
			}
			if res.Body.Len() == 0 {
				t.Fatal("empty dashboard response")
			}
		})
	}
}

func newTestHandler() http.Handler {
	return NewHandler(filepath.Join("..", "..", "dashboard"))
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
