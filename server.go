package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type adminResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type listResponse struct {
	List  []any `json:"list"`
	Total int   `json:"total"`
}

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthzHandler)
	mux.HandleFunc("/admin/", adminHandler)
	mux.HandleFunc("/admin", adminHandler)
	mux.Handle("/", http.FileServer(http.Dir("dashboard")))
	return mux
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAdminError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	data, ok := dashboardStartupData(r)
	if !ok {
		writeAdminError(w, http.StatusNotFound, "not found")
		return
	}
	writeAdminOK(w, data)
}

func dashboardStartupData(r *http.Request) (any, bool) {
	switch strings.TrimSuffix(r.URL.Path, "/") {
	case "/admin/stats/overview":
		return map[string]any{
			"request_count":     0,
			"success_count":     0,
			"error_count":       0,
			"total_tokens":      0,
			"total_cost":        "0.000000",
			"active_user_count": 0,
		}, true
	case "/admin/stats/daily", "/admin/channels", "/admin/usage-logs", "/admin/users", "/admin/rate-limits", "/admin/models":
		parsePagination(r)
		return listResponse{List: []any{}, Total: 0}, true
	case "/admin/stats/channels":
		return map[string]any{"list": []any{}}, true
	default:
		return nil, false
	}
}

func parsePagination(r *http.Request) (int, int) {
	q := r.URL.Query()
	page := parsePositiveInt(q.Get("page"), 1)
	pageSize := parsePositiveInt(q.Get("page_size"), 20)
	return page, pageSize
}

func parsePositiveInt(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func writeAdminOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, adminResponse{Code: 0, Message: "ok", Data: data})
}

func writeAdminError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, adminResponse{Code: status, Message: message, Data: map[string]any{}})
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
