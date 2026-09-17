package accounts

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"LLMGateway/server/internal/store"
)

type Server struct{ store store.Store }

func New(st store.Store) *Server { return &Server{store: st} }

func (a *Server) result(data any, err error) (any, bool, int, string) {
	if err == nil {
		return data, true, 0, ""
	}
	status := http.StatusInternalServerError
	if errors.Is(err, store.ErrNotFound) {
		status = http.StatusNotFound
	}
	if errors.Is(err, store.ErrInvalid) {
		status = http.StatusBadRequest
	}
	if errors.Is(err, store.ErrNotImplemented) {
		status = http.StatusNotImplemented
	}
	return nil, true, status, strings.TrimPrefix(err.Error(), store.ErrInvalid.Error()+": ")
}

func (a *Server) noBody(err error) (any, bool, int, string) {
	if err != nil {
		return a.result(nil, err)
	}
	return map[string]any{"deleted": true}, true, 0, ""
}

func readJSON(r *http.Request, value any) error { return json.NewDecoder(r.Body).Decode(value) }

func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		return nil
	}
	return parts
}

func ParsePagination(r *http.Request) (int, int) {
	q := r.URL.Query()
	return positiveInt(q.Get("page"), 1), positiveInt(q.Get("page_size"), 20)
}

func positiveInt(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
