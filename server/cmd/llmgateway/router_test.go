package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"LLMGateway/server/internal/testutil/storefake"
)

// TestNewRouterPaths exercises the production route table directly so drift
// between cmd/llmgateway/router.go and the handler entry points is caught.
func TestNewRouterPaths(t *testing.T) {
	dashboardDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dashboardDir, "index.html"), []byte("<!doctype html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	router := newRouter(dashboardDir, storefake.New())

	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"healthz", http.MethodGet, "/healthz", http.StatusOK},
		{"admin stats overview", http.MethodGet, "/admin/stats/overview", http.StatusOK},
		{"admin channels", http.MethodGet, "/admin/channels", http.StatusOK},
		{"v1 models requires auth", http.MethodGet, "/v1/models", http.StatusUnauthorized},
		{"v1 chat requires auth", http.MethodPost, "/v1/chat/completions", http.StatusUnauthorized},
		{"dashboard index", http.MethodGet, "/dashboard/index.html", http.StatusOK},
		{"dashboard client route", http.MethodGet, "/dashboard/channels", http.StatusOK},
		{"dashboard nested client route", http.MethodGet, "/dashboard/users/keys", http.StatusOK},
		{"root static", http.MethodGet, "/", http.StatusOK},
		{"unknown path", http.MethodGet, "/does-not-exist", http.StatusNotFound},
		{"unknown admin path", http.MethodGet, "/admin/does-not-exist", http.StatusNotFound},
		{"unknown v1 path", http.MethodGet, "/v1/does-not-exist", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)

			if res.Code != tt.want {
				t.Fatalf("%s %s status = %d, want %d; body=%s", tt.method, tt.path, res.Code, tt.want, res.Body.String())
			}
		})
	}
}
