package accounts

import (
	"net/http"

	"LLMGateway/server/internal/httpcommon"
)

func (a *Server) listUsers(r *http.Request) httpcommon.AdminResult {
	page, pageSize := httpcommon.ParsePagination(r)
	return httpcommon.Result(a.store.ListUsers(r.Context(), page, pageSize))
}

func (a *Server) createUser(r *http.Request) httpcommon.AdminResult {
	var req UserInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.CreateUser(r.Context(), req))
}

func (a *Server) updateUser(r *http.Request, id int) httpcommon.AdminResult {
	var req UserInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.UpdateUser(r.Context(), id, req))
}

func (a *Server) updateUserStatus(r *http.Request, id int) httpcommon.AdminResult {
	var req UserStatusInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.UpdateUserStatus(r.Context(), id, req.Status))
}

func (a *Server) rechargeUser(r *http.Request, id int) httpcommon.AdminResult {
	var req RechargeInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.RechargeUser(r.Context(), id, req))
}

func (a *Server) listBalanceTransactions(r *http.Request, id int) httpcommon.AdminResult {
	page, pageSize := httpcommon.ParsePagination(r)
	return httpcommon.Result(a.store.ListBalanceTransactions(r.Context(), id, page, pageSize))
}
