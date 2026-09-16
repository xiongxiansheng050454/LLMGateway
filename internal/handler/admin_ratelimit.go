package handler

import (
	"net/http"
	"strconv"
	"strings"

	"LLMGateway/internal/domain"
)

func (a *app) rateLimitData(r *http.Request) (any, bool, int, string) {
	parts := splitPath(strings.TrimSuffix(r.URL.Path, "/"))
	if len(parts) < 2 || parts[0] != "admin" || parts[1] != "rate-limits" {
		return nil, false, 0, ""
	}

	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			return a.listRateLimits(r)
		case http.MethodPost:
			return a.createRateLimit(r)
		}
		return nil, true, http.StatusMethodNotAllowed, "method not allowed"
	}

	id, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, true, http.StatusBadRequest, "invalid rule id"
	}
	if len(parts) == 3 {
		switch r.Method {
		case http.MethodPut:
			return a.updateRateLimit(r, id)
		case http.MethodDelete:
			return a.noBody(a.store.DeleteRateLimit(id))
		}
		return nil, true, http.StatusMethodNotAllowed, "method not allowed"
	}
	return nil, false, 0, ""
}

func (a *app) listRateLimits(r *http.Request) (any, bool, int, string) {
	page, pageSize := ParsePagination(r)

	var enabled *bool
	switch r.URL.Query().Get("enabled") {
	case "true":
		value := true
		enabled = &value
	case "false":
		value := false
		enabled = &value
	}
	return a.result(a.store.ListRateLimits(enabled, page, pageSize))
}

func (a *app) createRateLimit(r *http.Request) (any, bool, int, string) {
	var req domain.RateLimitInput
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return a.result(a.store.CreateRateLimit(req))
}

func (a *app) updateRateLimit(r *http.Request, id int) (any, bool, int, string) {
	var req domain.RateLimitInput
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return a.result(a.store.UpdateRateLimit(id, req))
}
