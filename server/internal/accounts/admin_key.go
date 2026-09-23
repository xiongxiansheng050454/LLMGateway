package accounts

import (
	"net/http"

	"LLMGateway/server/internal/httpcommon"
)

func (a *Server) listKeys(r *http.Request) httpcommon.AdminResult {
	page, pageSize := httpcommon.ParsePagination(r)
	return httpcommon.Result(a.store.ListKeys(page, pageSize))
}

func (a *Server) listUserKeys(r *http.Request, userID int) httpcommon.AdminResult {
	page, pageSize := httpcommon.ParsePagination(r)
	return httpcommon.Result(a.store.ListUserKeys(userID, page, pageSize))
}

func (a *Server) createKey(r *http.Request, userID int) httpcommon.AdminResult {
	var req KeyInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.CreateKey(userID, req))
}

func (a *Server) updateKey(r *http.Request, userID, keyID int) httpcommon.AdminResult {
	var req KeyUpdateInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.UpdateKey(userID, keyID, req))
}
