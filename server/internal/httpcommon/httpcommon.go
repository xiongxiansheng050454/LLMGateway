package httpcommon

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	apperrors "LLMGateway/server/internal/errors"
)

// ReadJSON decodes a request body into value.
func ReadJSON(r *http.Request, value any) error { return json.NewDecoder(r.Body).Decode(value) }

// SplitPath returns non-empty path components.
func SplitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		return nil
	}
	return parts
}

// ParsePagination reads the shared admin list pagination parameters.
func ParsePagination(r *http.Request) (int, int) {
	q := r.URL.Query()
	return positiveInt(q.Get("page"), 1), positiveInt(q.Get("page_size"), 20)
}

// Result converts a store operation into the business module handler result.
func Result(data any, err error) (any, bool, int, string) {
	if err == nil {
		return data, true, 0, ""
	}
	return nil, true, StatusFor(err), MessageFor(err)
}

// NoBody returns the common deletion response.
func NoBody(err error) (any, bool, int, string) {
	if err != nil {
		return Result(nil, err)
	}
	return map[string]any{"deleted": true}, true, 0, ""
}

func StatusFor(err error) int {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, apperrors.ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(err, apperrors.ErrNotImplemented):
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

func MessageFor(err error) string {
	return strings.TrimPrefix(err.Error(), apperrors.ErrInvalid.Error()+": ")
}

func positiveInt(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
