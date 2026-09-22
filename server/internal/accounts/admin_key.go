package accounts

import (
	"net/http"

	"LLMGateway/server/internal/httpcommon"
)

func (a *Server) listKeys(r *http.Request) (any, bool, int, string) {
	page, pageSize := httpcommon.ParsePagination(r)
	return httpcommon.Result(a.store.ListKeys(page, pageSize))
}

func (a *Server) listUserKeys(r *http.Request, userID int) (any, bool, int, string) {
	page, pageSize := httpcommon.ParsePagination(r)
	return httpcommon.Result(a.store.ListUserKeys(userID, page, pageSize))
}

func (a *Server) createKey(r *http.Request, userID int) (any, bool, int, string) {
	var req KeyInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return httpcommon.Result(a.CreateKey(userID, req))
}

func (a *Server) updateKey(r *http.Request, userID, keyID int) (any, bool, int, string) {
	var req KeyUpdateInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return httpcommon.Result(a.UpdateKey(userID, keyID, req))
}
