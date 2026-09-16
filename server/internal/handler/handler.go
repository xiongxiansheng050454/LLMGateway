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

	"LLMGateway/server/internal/store"
)

// Server holds the HTTP entry points for the gateway. The concrete route table
// (which path maps to which entry point) lives in cmd/llmgateway/router.go.
type Server struct {
	store        store.Store
	client       *http.Client
	randIntN     func(int) int
	now          func() time.Time
	dashboardDir string
	dashboard    http.Handler
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

// NewServer builds the HTTP entry points around an injected store. Route
// registration is done by cmd/llmgateway/router.go.
func NewServer(dashboardDir string, st store.Store, opts ...Option) *Server {
	settings := options{
		upstreamTimeout: 60 * time.Second,
		randIntN:        rand.Intn,
		now:             time.Now,
	}
	for _, opt := range opts {
		opt(&settings)
	}
	return &Server{
		store:        st,
		client:       &http.Client{Timeout: settings.upstreamTimeout},
		randIntN:     settings.randIntN,
		now:          settings.now,
		dashboardDir: dashboardDir,
		dashboard:    http.FileServer(http.Dir(dashboardDir)),
	}
}

// Dashboard serves the static dashboard directory.
func (a *Server) Dashboard(w http.ResponseWriter, r *http.Request) {
	a.dashboard.ServeHTTP(w, r)
}

// DashboardIndex serves the dashboard index page directly.
func (a *Server) DashboardIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeMethodNotAllowed(w)
		return
	}

	file, err := os.Open(filepath.Join(a.dashboardDir, "index.html"))
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

// Healthz reports service health.
func (a *Server) Healthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Admin handles the /admin management API.
func (a *Server) Admin(w http.ResponseWriter, r *http.Request) {
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

func (a *Server) adminData(r *http.Request) (any, bool, int, string) {
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
func (a *Server) catalogData(r *http.Request) (any, bool, int, string) {
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

func (a *Server) result(data any, err error) (any, bool, int, string) {
	if err != nil {
		return errorResponse(err)
	}
	return data, true, 0, ""
}

func (a *Server) noBody(err error) (any, bool, int, string) {
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
