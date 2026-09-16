package handler

import (
	"net/http"

	"LLMGateway/internal/domain"
)

func (a *app) listKeys(r *http.Request) (any, bool, int, string) {
	page, pageSize := ParsePagination(r)
	return a.result(a.store.ListKeys(page, pageSize))
}

func (a *app) listUserKeys(r *http.Request, userID int) (any, bool, int, string) {
	page, pageSize := ParsePagination(r)
	return a.result(a.store.ListUserKeys(userID, page, pageSize))
}

func (a *app) createKey(r *http.Request, userID int) (any, bool, int, string) {
	var req domain.KeyInput
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return a.result(a.store.CreateKey(userID, req))
}

func (a *app) updateKey(r *http.Request, userID, keyID int) (any, bool, int, string) {
	var req domain.KeyUpdateInput
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return a.result(a.store.UpdateKey(userID, keyID, req))
}
