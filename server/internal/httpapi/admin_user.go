package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"LLMGateway/server/internal/domain"
)

// userData dispatches /admin/users and /admin/keys requests.
func (a *Server) userData(r *http.Request) (any, bool, int, string) {
	parts := splitPath(strings.TrimSuffix(r.URL.Path, "/"))
	if len(parts) < 2 || parts[0] != "admin" {
		return nil, false, 0, ""
	}

	if parts[1] == "keys" && len(parts) == 2 {
		if r.Method == http.MethodGet {
			return a.listKeys(r)
		}
		return nil, true, http.StatusMethodNotAllowed, "method not allowed"
	}
	if parts[1] != "users" {
		return nil, false, 0, ""
	}
	return a.userRoutes(r, parts)
}

func (a *Server) userRoutes(r *http.Request, parts []string) (any, bool, int, string) {
	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			return a.listUsers(r)
		case http.MethodPost:
			return a.createUser(r)
		}
		return nil, true, http.StatusMethodNotAllowed, "method not allowed"
	}

	userID, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, true, http.StatusBadRequest, "invalid user id"
	}

	if len(parts) == 3 {
		switch r.Method {
		case http.MethodPut:
			return a.updateUser(r, userID)
		case http.MethodDelete:
			return a.noBody(a.store.DeleteUser(userID))
		}
		return nil, true, http.StatusMethodNotAllowed, "method not allowed"
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
				return a.result(a.store.GetUserBalance(userID))
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
		return nil, true, http.StatusMethodNotAllowed, "method not allowed"
	}

	if len(parts) == 5 && parts[3] == "keys" {
		keyID, err := strconv.Atoi(parts[4])
		if err != nil {
			return nil, true, http.StatusBadRequest, "invalid key id"
		}
		switch r.Method {
		case http.MethodPut:
			return a.updateKey(r, userID, keyID)
		case http.MethodDelete:
			return a.noBody(a.store.DeleteKey(userID, keyID))
		}
		return nil, true, http.StatusMethodNotAllowed, "method not allowed"
	}

	if len(parts) == 6 && parts[3] == "keys" && parts[5] == "reset" {
		keyID, err := strconv.Atoi(parts[4])
		if err != nil {
			return nil, true, http.StatusBadRequest, "invalid key id"
		}
		if r.Method == http.MethodPost {
			return a.result(a.store.ResetKey(userID, keyID))
		}
		return nil, true, http.StatusMethodNotAllowed, "method not allowed"
	}

	return nil, false, 0, ""
}

func (a *Server) listUsers(r *http.Request) (any, bool, int, string) {
	page, pageSize := ParsePagination(r)
	return a.result(a.store.ListUsers(page, pageSize))
}

func (a *Server) createUser(r *http.Request) (any, bool, int, string) {
	var req domain.UserInput
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return a.result(a.store.CreateUser(req))
}

func (a *Server) updateUser(r *http.Request, id int) (any, bool, int, string) {
	var req domain.UserInput
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return a.result(a.store.UpdateUser(id, req))
}

func (a *Server) updateUserStatus(r *http.Request, id int) (any, bool, int, string) {
	var req domain.UserStatusInput
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return a.result(a.store.UpdateUserStatus(id, req.Status))
}

func (a *Server) rechargeUser(r *http.Request, id int) (any, bool, int, string) {
	var req domain.RechargeInput
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return a.result(a.store.RechargeUser(id, req))
}

func (a *Server) listBalanceTransactions(r *http.Request, id int) (any, bool, int, string) {
	page, pageSize := ParsePagination(r)
	return a.result(a.store.ListBalanceTransactions(id, page, pageSize))
}
