package handler

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"LLMGateway/internal/store"
)

type app struct {
	store    store.Store
	client   *http.Client
	randIntN func(int) int
	now      func() time.Time
}

type options struct {
	upstreamTimeout time.Duration
	randIntN        func(int) int
	now             func() time.Time
}

// Option customizes handler assembly.
type Option func(*options)

// WithUpstreamTimeout sets the HTTP client timeout for downstream proxy calls.
// The default is intentionally longer than the admin timeout.
func WithUpstreamTimeout(timeout time.Duration) Option {
	return func(o *options) {
		if timeout > 0 {
			o.upstreamTimeout = timeout
		}
	}
}

// WithRandSource injects the random source used for weighted routing so tests
// can make selection deterministic.
func WithRandSource(fn func(int) int) Option {
	return func(o *options) {
		if fn != nil {
			o.randIntN = fn
		}
	}
}

// WithClock injects the clock used for key expiry and rate limit windows.
func WithClock(fn func() time.Time) Option {
	return func(o *options) {
		if fn != nil {
			o.now = fn
		}
	}
}

// NewHandlerWithStore builds the HTTP handler around an injected store.
// Store assembly belongs to the process composition root (cmd/llmgateway).
func NewHandlerWithStore(dashboardDir string, st store.Store, opts ...Option) http.Handler {
	settings := options{
		upstreamTimeout: 60 * time.Second,
		randIntN:        rand.Intn,
		now:             time.Now,
	}
	for _, opt := range opts {
		opt(&settings)
	}
	client := &http.Client{Timeout: settings.upstreamTimeout}
	a := &app{
		store:    st,
		client:   client,
		randIntN: settings.randIntN,
		now:      settings.now,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthzHandler)
	mux.HandleFunc("/admin/", a.adminHandler)
	mux.HandleFunc("/admin", a.adminHandler)
	mux.HandleFunc("/v1/", a.openaiHandler)
	mux.HandleFunc("/v1", a.openaiHandler)

	dashboard := http.FileServer(http.Dir(dashboardDir))
	mux.HandleFunc("/dashboard/index.html", dashboardIndexHandler(dashboardDir))
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard", dashboard))
	mux.Handle("/", dashboard)
	return mux
}

func dashboardIndexHandler(dashboardDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeMethodNotAllowed(w)
			return
		}

		file, err := os.Open(filepath.Join(dashboardDir, "index.html"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, "index.html", info.ModTime(), file)
	}
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *app) adminHandler(w http.ResponseWriter, r *http.Request) {
	writeCORSHeaders(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	data, ok, errStatus, errMsg := a.adminData(r)
	if errStatus != 0 {
		writeAdminError(w, errStatus, errMsg)
		return
	}
	if !ok {
		writeAdminError(w, http.StatusNotFound, "not found")
		return
	}
	writeAdminOK(w, data)
}

func (a *app) adminData(r *http.Request) (any, bool, int, string) {
	if data, ok, status, msg := a.catalogData(r); ok || status != 0 {
		return data, ok, status, msg
	}
	if data, ok, status, msg := a.userData(r); ok || status != 0 {
		return data, ok, status, msg
	}
	if data, ok, status, msg := a.rateLimitData(r); ok || status != 0 {
		return data, ok, status, msg
	}
	if data, ok, status, msg := a.usageData(r); ok || status != 0 {
		return data, ok, status, msg
	}
	return nil, false, 0, ""
}

// catalogData dispatches /admin requests to the channel, model and pricing
// domain handlers.
func (a *app) catalogData(r *http.Request) (any, bool, int, string) {
	parts := splitPath(strings.TrimSuffix(r.URL.Path, "/"))
	if len(parts) < 2 || parts[0] != "admin" {
		return nil, false, 0, ""
	}
	if parts[1] == "channels" {
		return a.channelData(r, parts)
	}
	if parts[1] == "models" && len(parts) == 2 && r.Method == http.MethodGet {
		return a.result(a.store.ListCatalogModels(r.URL.Query().Get("status") == "1"))
	}
	if parts[1] == "pricing" && len(parts) == 2 {
		return a.pricingData(r)
	}
	return nil, false, 0, ""
}

func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		return nil
	}
	return parts
}

func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func ParsePagination(r *http.Request) (int, int) {
	q := r.URL.Query()
	page := parsePositiveInt(q.Get("page"), 1)
	pageSize := parsePositiveInt(q.Get("page_size"), 20)
	return page, pageSize
}

func parsePositiveInt(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func (a *app) result(data any, err error) (any, bool, int, string) {
	if err != nil {
		return errorResponse(err)
	}
	return data, true, 0, ""
}

func (a *app) noBody(err error) (any, bool, int, string) {
	if err != nil {
		return errorResponse(err)
	}
	return map[string]any{"deleted": true}, true, 0, ""
}

func errorResponse(err error) (any, bool, int, string) {
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

func writeAdminOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, adminResponse{Code: 0, Message: "ok", Data: data})
}

func writeAdminError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, adminResponse{Code: status, Message: message, Data: map[string]any{}})
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func writeCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type")
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
