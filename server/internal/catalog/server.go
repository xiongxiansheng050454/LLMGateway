package catalog

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"LLMGateway/server/internal/store"
)

type Server struct {
	store  store.Store
	client *http.Client
}

func New(st store.Store, client *http.Client) *Server { return &Server{store: st, client: client} }

func (a *Server) result(data any, err error) (any, bool, int, string) {
	if err == nil {
		return data, true, 0, ""
	}
	return nil, true, statusFor(err), messageFor(err)
}

func (a *Server) noBody(err error) (any, bool, int, string) {
	if err != nil {
		return a.result(nil, err)
	}
	return map[string]any{"deleted": true}, true, 0, ""
}

func statusFor(err error) int {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, store.ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(err, store.ErrNotImplemented):
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

func messageFor(err error) string {
	return strings.TrimPrefix(err.Error(), store.ErrInvalid.Error()+": ")
}

func readJSON(r *http.Request, value any) error { return json.NewDecoder(r.Body).Decode(value) }

func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		return nil
	}
	return parts
}
