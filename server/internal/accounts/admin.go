package accounts

import (
	"net/http"
	"strings"

	"LLMGateway/server/internal/httpcommon"
)

// Data dispatches account-related /admin/users and /admin/keys requests.
func (a *Server) Data(r *http.Request) httpcommon.AdminResult {
	parts := httpcommon.SplitPath(strings.TrimSuffix(r.URL.Path, "/"))
	if len(parts) < 2 || parts[0] != "admin" {
		return httpcommon.Unhandled()
	}

	if parts[1] == "keys" || (parts[1] == "users" && len(parts) >= 4 && parts[3] == "keys") {
		return a.keyRoutes(r, parts)
	}
	if parts[1] != "users" {
		return httpcommon.Unhandled()
	}
	return a.userRoutes(r, parts)
}
