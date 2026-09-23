package quota

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"LLMGateway/server/internal/httpcommon"
)

func (a *Server) Data(r *http.Request) httpcommon.AdminResult {
	parts := httpcommon.SplitPath(strings.TrimSuffix(r.URL.Path, "/"))
	if len(parts) < 2 || parts[0] != "admin" {
		return httpcommon.Unhandled()
	}
	if parts[1] == "quota-usage" && len(parts) == 2 {
		if r.Method != http.MethodGet {
			return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
		}
		return a.listUsage(r)
	}
	if parts[1] != "quota-policies" {
		return httpcommon.Unhandled()
	}
	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			return a.listPolicies(r)
		case http.MethodPost:
			return a.createPolicy(r)
		default:
			return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
		}
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid policy id")
	}
	switch r.Method {
	case http.MethodPut:
		return a.updatePolicy(r, id)
	case http.MethodDelete:
		return httpcommon.NoBody(a.DeleteQuotaPolicy(id))
	default:
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *Server) listPolicies(r *http.Request) httpcommon.AdminResult {
	page, pageSize := httpcommon.ParsePagination(r)
	filter := QuotaPolicyFilter{ScopeType: r.URL.Query().Get("scope_type"), Page: page, PageSize: pageSize}
	filter.ScopeID, _ = strconv.Atoi(r.URL.Query().Get("scope_id"))
	if value := r.URL.Query().Get("enabled"); value == "true" || value == "false" {
		enabled := value == "true"
		filter.Enabled = &enabled
	}
	return httpcommon.Result(a.ListQuotaPolicies(filter))
}

func (a *Server) listUsage(r *http.Request) httpcommon.AdminResult {
	page, pageSize := httpcommon.ParsePagination(r)
	filter := QuotaPolicyFilter{ScopeType: r.URL.Query().Get("scope_type"), Page: page, PageSize: pageSize}
	filter.ScopeID, _ = strconv.Atoi(r.URL.Query().Get("scope_id"))
	return httpcommon.Result(a.ListQuotaUsage(context.Background(), filter))
}

func (a *Server) createPolicy(r *http.Request) httpcommon.AdminResult {
	var input QuotaPolicyInput
	if err := httpcommon.ReadJSON(r, &input); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.CreateQuotaPolicy(input))
}

func (a *Server) updatePolicy(r *http.Request, id int) httpcommon.AdminResult {
	var input QuotaPolicyInput
	if err := httpcommon.ReadJSON(r, &input); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.UpdateQuotaPolicy(id, input))
}
