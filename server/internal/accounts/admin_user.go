package accounts

import (
	"net/http"
	"strconv"
	"strings"

	"LLMGateway/server/internal/httpcommon"
)

// Data userData dispatches /admin/users and /admin/keys requests.
func (a *Server) Data(r *http.Request) httpcommon.AdminResult {
	parts := httpcommon.SplitPath(strings.TrimSuffix(r.URL.Path, "/"))
	if len(parts) < 2 || parts[0] != "admin" {
		return httpcommon.Unhandled()
	}

	if parts[1] == "keys" && len(parts) == 2 {
		if r.Method == http.MethodGet {
			return a.listKeys(r)
		}
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
	if parts[1] != "users" {
		return httpcommon.Unhandled()
	}
	return a.userRoutes(r, parts)
}

func (a *Server) userRoutes(r *http.Request, parts []string) httpcommon.AdminResult {
	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			return a.listUsers(r)
		case http.MethodPost:
			return a.createUser(r)
		}
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}

	userID, err := strconv.Atoi(parts[2])
	if err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid user id")
	}

	if len(parts) == 3 {
		switch r.Method {
		case http.MethodPut:
			return a.updateUser(r, userID)
		case http.MethodDelete:
			return httpcommon.NoBody(a.DeleteUser(userID))
		}
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}

	if len(parts) == 4 {
		switch parts[3] {
		case "status":
			if r.Method == http.MethodPut {
				return a.updateUserStatus(r, userID)
			}
		case "recharge":
			if r.Method == http.MethodPost {
				return a.rechargeUser(r, userID)
			}
		case "balance":
			if r.Method == http.MethodGet {
				return httpcommon.Result(a.store.GetUserBalance(userID))
			}
		case "balance-transactions":
			if r.Method == http.MethodGet {
				return a.listBalanceTransactions(r, userID)
			}
		case "keys":
			switch r.Method {
			case http.MethodGet:
				return a.listUserKeys(r, userID)
			case http.MethodPost:
				return a.createKey(r, userID)
			}
		}
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}

	if len(parts) == 5 && parts[3] == "keys" {
		keyID, err := strconv.Atoi(parts[4])
		if err != nil {
			return httpcommon.HTTPError(http.StatusBadRequest, "invalid key id")
		}
		switch r.Method {
		case http.MethodPut:
			return a.updateKey(r, userID, keyID)
		case http.MethodDelete:
			return httpcommon.NoBody(a.DeleteKey(userID, keyID))
		}
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}

	if len(parts) == 6 && parts[3] == "keys" && parts[5] == "reset" {
		keyID, err := strconv.Atoi(parts[4])
		if err != nil {
			return httpcommon.HTTPError(http.StatusBadRequest, "invalid key id")
		}
		if r.Method == http.MethodPost {
			return httpcommon.Result(a.ResetKey(userID, keyID))
		}
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}

	return httpcommon.Unhandled()
}

func (a *Server) listUsers(r *http.Request) httpcommon.AdminResult {
	page, pageSize := httpcommon.ParsePagination(r)
	return httpcommon.Result(a.store.ListUsers(page, pageSize))
}

func (a *Server) createUser(r *http.Request) httpcommon.AdminResult {
	var req UserInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.CreateUser(req))
}

func (a *Server) updateUser(r *http.Request, id int) httpcommon.AdminResult {
	var req UserInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.UpdateUser(id, req))
}

func (a *Server) updateUserStatus(r *http.Request, id int) httpcommon.AdminResult {
	var req UserStatusInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.UpdateUserStatus(id, req.Status))
}

func (a *Server) rechargeUser(r *http.Request, id int) httpcommon.AdminResult {
	var req RechargeInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.RechargeUser(id, req))
}

func (a *Server) listBalanceTransactions(r *http.Request, id int) httpcommon.AdminResult {
	page, pageSize := httpcommon.ParsePagination(r)
	return httpcommon.Result(a.store.ListBalanceTransactions(id, page, pageSize))
}
