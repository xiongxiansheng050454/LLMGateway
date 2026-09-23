package ratelimit

import (
	"net/http"
	"strconv"
	"strings"

	"LLMGateway/server/internal/httpcommon"
)

func (a *Server) Data(r *http.Request) httpcommon.AdminResult {
	parts := httpcommon.SplitPath(strings.TrimSuffix(r.URL.Path, "/"))
	if len(parts) < 2 || parts[0] != "admin" || parts[1] != "rate-limits" {
		return httpcommon.Unhandled()
	}

	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			return a.listRateLimits(r)
		case http.MethodPost:
			return a.createRateLimit(r)
		}
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}

	id, err := strconv.Atoi(parts[2])
	if err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid rule id")
	}
	if len(parts) == 3 {
		switch r.Method {
		case http.MethodPut:
			return a.updateRateLimit(r, id)
		case http.MethodDelete:
			return httpcommon.NoBody(a.DeleteRateLimit(id))
		}
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
	return httpcommon.Unhandled()
}

func (a *Server) listRateLimits(r *http.Request) httpcommon.AdminResult {
	page, pageSize := httpcommon.ParsePagination(r)

	var enabled *bool
	switch r.URL.Query().Get("enabled") {
	case "true":
		value := true
		enabled = &value
	case "false":
		value := false
		enabled = &value
	}
	return httpcommon.Result(a.ListRateLimits(enabled, page, pageSize))
}

func (a *Server) createRateLimit(r *http.Request) httpcommon.AdminResult {
	var req RateLimitInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.CreateRateLimit(req))
}

func (a *Server) updateRateLimit(r *http.Request, id int) httpcommon.AdminResult {
	var req RateLimitInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.UpdateRateLimit(id, req))
}
