package httpapi

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"LLMGateway/server/internal/accounts"
	"LLMGateway/server/internal/catalog"
	"LLMGateway/server/internal/httpcommon"
	"LLMGateway/server/internal/proxy"
	openaiwire "LLMGateway/server/internal/proxy/openai"
	"LLMGateway/server/internal/quota"
	"LLMGateway/server/internal/ratelimit"
	"LLMGateway/server/internal/usage"
)

// Server holds the HTTP entry points for the gateway. The concrete route table
// (which path maps to which entry point) lives in cmd/llmgateway/router.go.
// adminResponse is the unified envelope for /admin endpoints.
type adminResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type Server struct {
	store        Port
	client       *http.Client
	proxy        *proxy.Service
	catalog      *catalog.Server
	accounts     *accounts.Server
	usage        *usage.Server
	ratelimit    *ratelimit.Server
	quota        *quota.Server
	dashboardDir string
	dashboard    http.Handler
}

// Port is the process assembly contract. Each business server receives its
// own narrower module port during construction.
//
// Transaction managers are exposed as per-module factory methods because a
// single store cannot implement several InTx methods that differ only in the
// callback signature.
type Port interface {
	accounts.Port
	catalog.Port
	catalog.HealthPort
	quota.Port
	ratelimit.Port
	usage.Port
	proxy.Port
	AccountsTx() accounts.TxManager
}

type options struct {
	upstreamTimeout       time.Duration
	randIntN              func(int) int
	now                   func() time.Time
	quotaDefaultMaxTokens int
	quotaReservationTTL   time.Duration
	upstreamMaxAttempts   int
	minimumRouteBalance   string
}

func WithQuotaConfig(defaultMaxTokens int, reservationTTL time.Duration) Option {
	return func(o *options) {
		o.quotaDefaultMaxTokens = defaultMaxTokens
		o.quotaReservationTTL = reservationTTL
	}
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

func WithUpstreamMaxAttempts(attempts int) Option {
	return func(o *options) {
		if attempts > 0 {
			o.upstreamMaxAttempts = attempts
		}
	}
}

// WithMinimumRouteBalance sets the global reserve below which chargeable
// channels are excluded from routing.
func WithMinimumRouteBalance(balance string) Option {
	return func(o *options) {
		o.minimumRouteBalance = balance
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
func NewServer(dashboardDir string, st Port, opts ...Option) *Server {
	settings := options{
		upstreamTimeout:       60 * time.Second,
		randIntN:              rand.Intn,
		now:                   time.Now,
		quotaDefaultMaxTokens: 4096,
		quotaReservationTTL:   2 * time.Minute,
		upstreamMaxAttempts:   3,
	}
	for _, opt := range opts {
		opt(&settings)
	}
	client := &http.Client{Timeout: settings.upstreamTimeout}
	proxyService := proxy.NewService(st, client, settings.randIntN, settings.now, openaiwire.Adapter())
	proxyService.ConfigureQuota(settings.quotaDefaultMaxTokens, settings.quotaReservationTTL)
	proxyService.ConfigureRequest(settings.upstreamTimeout, settings.upstreamMaxAttempts)
	proxyService.ConfigureMinimumRouteBalance(settings.minimumRouteBalance)
	return &Server{
		store:        st,
		client:       client,
		proxy:        proxyService,
		catalog:      catalog.New(st, client),
		accounts:     accounts.New(st, st.AccountsTx()),
		usage:        usage.New(st),
		ratelimit:    ratelimit.New(st),
		quota:        quota.New(st),
		dashboardDir: dashboardDir,
		dashboard:    http.FileServer(http.Dir(dashboardDir)),
	}
}

// Dashboard serves static dashboard assets and falls back to index.html for
// client-side routes handled by the React BrowserRouter.
func (a *Server) Dashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeMethodNotAllowed(w)
		return
	}

	path := strings.TrimPrefix(filepath.Clean(r.URL.Path), string(filepath.Separator))
	if path != "." && path != "" {
		if file, err := os.Open(filepath.Join(a.dashboardDir, filepath.FromSlash(path))); err == nil {
			file.Close()
			a.dashboard.ServeHTTP(w, r)
			return
		}
	}
	a.DashboardIndex(w, r)
}

// DashboardRoot serves the legacy root mount without treating arbitrary root
// paths as client-side dashboard routes.
func (a *Server) DashboardRoot(w http.ResponseWriter, r *http.Request) {
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
	parts := httpcommon.SplitPath(strings.TrimSuffix(r.URL.Path, "/"))
	if data, ok, status, msg := a.catalog.Data(r, parts); ok || status != 0 {
		return data, ok, status, msg
	}
	if data, ok, status, msg := a.accounts.Data(r); ok || status != 0 {
		return data, ok, status, msg
	}
	if data, ok, status, msg := a.ratelimit.Data(r); ok || status != 0 {
		return data, ok, status, msg
	}
	if data, ok, status, msg := a.quota.Data(r); ok || status != 0 {
		return data, ok, status, msg
	}
	if data, ok, status, msg := a.usage.Data(r); ok || status != 0 {
		return data, ok, status, msg
	}
	return nil, false, 0, ""
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
