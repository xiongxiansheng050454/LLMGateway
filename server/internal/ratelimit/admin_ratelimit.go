package ratelimit

import (
	"net/http"
	"strconv"

	"LLMGateway/server/internal/httpcommon"
)

func (a *Server) RegisterAdminRoutes(mux *http.ServeMux) {
	httpcommon.HandleAdmin(mux, "/admin/rate-limits", a.rateLimits)
	httpcommon.HandleAdmin(mux, "/admin/rate-limits/{id}", a.rateLimit)
}

func (a *Server) rateLimits(r *http.Request) httpcommon.AdminResult {
	switch r.Method {
	case http.MethodGet:
		return a.listRateLimits(r)
	case http.MethodPost:
		return a.createRateLimit(r)
	default:
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *Server) rateLimit(r *http.Request) httpcommon.AdminResult {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid rule id")
	}
	switch r.Method {
	case http.MethodPut:
		return a.updateRateLimit(r, id)
	case http.MethodDelete:
		return httpcommon.NoBody(a.DeleteRateLimit(r.Context(), id))
	default:
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
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
	return httpcommon.Result(a.ListRateLimits(r.Context(), enabled, page, pageSize))
}

func (a *Server) createRateLimit(r *http.Request) httpcommon.AdminResult {
	var req RateLimitInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.CreateRateLimit(r.Context(), req))
}

func (a *Server) updateRateLimit(r *http.Request, id int) httpcommon.AdminResult {
	var req RateLimitInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.UpdateRateLimit(r.Context(), id, req))
}
