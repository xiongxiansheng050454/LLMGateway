package accounts

import (
	"net/http"
	"strconv"

	"LLMGateway/server/internal/httpcommon"
)

func (a *Server) keys(r *http.Request) httpcommon.AdminResult {
	if r.Method != http.MethodGet {
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
	return a.listKeys(r)
}

func (a *Server) userKeys(r *http.Request) httpcommon.AdminResult {
	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid user id")
	}
	switch r.Method {
	case http.MethodGet:
		return a.listUserKeys(r, userID)
	case http.MethodPost:
		return a.createKey(r, userID)
	default:
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *Server) userKey(r *http.Request) httpcommon.AdminResult {
	userID, keyID, result := a.keyPathIDs(r)
	if result.Status != 0 {
		return result
	}
	switch r.Method {
	case http.MethodPut:
		return a.updateKey(r, userID, keyID)
	case http.MethodDelete:
		return httpcommon.NoBody(a.DeleteKey(userID, keyID))
	default:
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *Server) userKeyReset(r *http.Request) httpcommon.AdminResult {
	if r.Method != http.MethodPost {
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
	userID, keyID, result := a.keyPathIDs(r)
	if result.Status != 0 {
		return result
	}
	return httpcommon.Result(a.ResetKey(userID, keyID))
}

func (a *Server) keyPathIDs(r *http.Request) (int, int, httpcommon.AdminResult) {
	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return 0, 0, httpcommon.HTTPError(http.StatusBadRequest, "invalid user id")
	}
	keyID, err := strconv.Atoi(r.PathValue("key_id"))
	if err != nil {
		return 0, 0, httpcommon.HTTPError(http.StatusBadRequest, "invalid key id")
	}
	return userID, keyID, httpcommon.AdminResult{}
}

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
