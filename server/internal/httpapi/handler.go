package httpapi

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"LLMGateway/server/internal/accounts"
	"LLMGateway/server/internal/catalog"
	"LLMGateway/server/internal/crypto"
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
	store     Port
	client    *http.Client
	proxy     *proxy.Service
	catalog   *catalog.Server
	accounts  *accounts.Server
	usage     *usage.Server
	ratelimit *ratelimit.Server
	quota     *quota.Server
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
	CatalogTx() catalog.TxManager
	QuotaTx() quota.TxManager
}

type options struct {
	upstreamTimeout       time.Duration
	randIntN              func(int) int
	now                   func() time.Time
	quotaDefaultMaxTokens int
	quotaReservationTTL   time.Duration
	upstreamMaxAttempts   int
	minimumRouteBalance   string
	cipher                *crypto.Cipher
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

// WithCipher injects the cipher used to encrypt/decrypt upstream channel keys.
func WithCipher(cipher *crypto.Cipher) Option {
	return func(o *options) {
		o.cipher = cipher
	}
}

// NewServer builds the HTTP entry points around an injected store. Route
// registration is done by cmd/llmgateway/router.go.
func NewServer(st Port, opts ...Option) *Server {
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
	catalogServer := catalog.New(catalog.Deps{
		Store:  st,
		Health: st,
		Tx:     st.CatalogTx(),
		Cipher: settings.cipher,
		Client: client,
		Now:    settings.now,
	})
	quotaServer := quota.New(st, st.QuotaTx(), settings.now)
	ratelimitServer := ratelimit.New(st, settings.now)
	proxyService := proxy.NewService(st, catalogServer, quotaServer, ratelimitServer, client, settings.randIntN, settings.now, openaiwire.Adapter())
	proxyService.ConfigureQuota(settings.quotaDefaultMaxTokens, settings.quotaReservationTTL)
	proxyService.ConfigureRequest(settings.upstreamTimeout, settings.upstreamMaxAttempts)
	proxyService.ConfigureMinimumRouteBalance(settings.minimumRouteBalance)
	return &Server{
		store:     st,
		client:    client,
		proxy:     proxyService,
		catalog:   catalogServer,
		accounts:  accounts.New(st, st.AccountsTx()),
		usage:     usage.New(st),
		ratelimit: ratelimitServer,
		quota:     quotaServer,
	}
}

// Healthy reports service health.
func (a *Server) Healthy(w http.ResponseWriter, r *http.Request) {
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

	result := a.adminData(r)
	if result.Status != 0 {
		writeAdminError(w, result.Status, result.Message)
		return
	}
	if !result.Handled {
		writeAdminError(w, http.StatusNotFound, "not found")
		return
	}
	writeAdminOK(w, result.Data)
}

func (a *Server) adminData(r *http.Request) httpcommon.AdminResult {
	parts := httpcommon.SplitPath(strings.TrimSuffix(r.URL.Path, "/"))
	if result := a.catalog.Data(r, parts); result.Handled || result.Status != 0 {
		return result
	}
	if result := a.accounts.Data(r); result.Handled || result.Status != 0 {
		return result
	}
	if result := a.ratelimit.Data(r); result.Handled || result.Status != 0 {
		return result
	}
	if result := a.quota.Data(r); result.Handled || result.Status != 0 {
		return result
	}
	if result := a.usage.Data(r); result.Handled || result.Status != 0 {
		return result
	}
	return httpcommon.Unhandled()
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
