package accounts

import (
	"net/http"
	"strconv"

	"LLMGateway/server/internal/httpcommon"
)

func (a *Server) keyRoutes(r *http.Request, parts []string) httpcommon.AdminResult {
	if len(parts) == 2 {
		if parts[1] == "keys" && r.Method == http.MethodGet {
			return a.listKeys(r)
		}
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}

	if len(parts) < 4 || parts[1] != "users" || parts[3] != "keys" {
		return httpcommon.Unhandled()
	}
	userID, err := strconv.Atoi(parts[2])
	if err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid user id")
	}

	if len(parts) == 4 {
		switch r.Method {
		case http.MethodGet:
			return a.listUserKeys(r, userID)
		case http.MethodPost:
			return a.createKey(r, userID)
		default:
			return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
		}
	}

	if len(parts) == 5 || (len(parts) == 6 && parts[5] == "reset") {
		keyID, err := strconv.Atoi(parts[4])
		if err != nil {
			return httpcommon.HTTPError(http.StatusBadRequest, "invalid key id")
		}
		if len(parts) == 6 {
			if r.Method == http.MethodPost {
				return httpcommon.Result(a.ResetKey(userID, keyID))
			}
			return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
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

	return httpcommon.Unhandled()
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
