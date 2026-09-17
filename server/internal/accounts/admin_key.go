package accounts

import (
	"net/http"

	"LLMGateway/server/internal/domain"
)

func (a *Server) listKeys(r *http.Request) (any, bool, int, string) {
	page, pageSize := ParsePagination(r)
	return a.result(a.store.ListKeys(page, pageSize))
}

func (a *Server) listUserKeys(r *http.Request, userID int) (any, bool, int, string) {
	page, pageSize := ParsePagination(r)
	return a.result(a.store.ListUserKeys(userID, page, pageSize))
}

func (a *Server) createKey(r *http.Request, userID int) (any, bool, int, string) {
	var req domain.KeyInput
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return a.result(a.store.CreateKey(userID, req))
}

func (a *Server) updateKey(r *http.Request, userID, keyID int) (any, bool, int, string) {
	var req domain.KeyUpdateInput
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return a.result(a.store.UpdateKey(userID, keyID, req))
}
