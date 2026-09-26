package catalog

import (
	"context"
	"net/http"
	"sync"
	"time"

	"LLMGateway/server/internal/crypto"
)

// breakerConfigCacheTTL bounds how long a per-channel breaker override is
// cached. Recording an attempt happens on the request path, so re-reading the
// override every attempt would add a query; admin writes invalidate eagerly.
const breakerConfigCacheTTL = 30 * time.Second

type breakerConfigCacheEntry struct {
	config  ChannelBreakerConfig
	expires time.Time
}

// Deps carries the catalog server's persistence primitives and runtime
// dependencies. Cipher is required for channel key encryption/decryption.
type Deps struct {
	Store   Port
	Health  HealthPort
	Tx      TxManager
	Cipher  *crypto.Cipher
	Client  *http.Client
	Now     func() time.Time
	Breaker ChannelBreakerConfig
}

// Server owns catalog rules and orchestration over injected primitives.
type Server struct {
	store       Port
	health      HealthPort
	tx          TxManager
	cipher      *crypto.Cipher
	client      *http.Client
	now         func() time.Time
	breaker     ChannelBreakerConfig
	testTimeout time.Duration

	breakerMu    sync.RWMutex
	breakerCache map[int]breakerConfigCacheEntry
}

func New(d Deps) *Server {
	now := d.Now
	if now == nil {
		now = time.Now
	}
	breaker := d.Breaker
	if breaker == (ChannelBreakerConfig{}) {
		breaker = DefaultChannelBreakerConfig()
	}
	return &Server{
		store:        d.Store,
		health:       d.Health,
		tx:           d.Tx,
		cipher:       d.Cipher,
		client:       d.Client,
		now:          now,
		breaker:      breaker,
		testTimeout:  channelTestTimeout,
		breakerCache: map[int]breakerConfigCacheEntry{},
	}
}

// breakerFor resolves the effective breaker config for a channel: the global
// default overlaid with any per-channel override. Overrides are cached briefly
// because recording runs on the proxy request path.
func (a *Server) breakerFor(ctx context.Context, channelID int) ChannelBreakerConfig {
	now := a.now()
	a.breakerMu.RLock()
	entry, ok := a.breakerCache[channelID]
	a.breakerMu.RUnlock()
	if ok && now.Before(entry.expires) {
		return entry.config
	}

	resolved := a.breaker
	override, found, err := a.health.GetChannelBreakerConfigRow(ctx, channelID)
	if err == nil && found {
		resolved = ResolveChannelBreakerConfig(a.breaker, &override)
	}
	a.breakerMu.Lock()
	a.breakerCache[channelID] = breakerConfigCacheEntry{config: resolved, expires: now.Add(breakerConfigCacheTTL)}
	a.breakerMu.Unlock()
	return resolved
}

// invalidateBreakerConfig drops the cached override after an admin write.
func (a *Server) invalidateBreakerConfig(channelID int) {
	a.breakerMu.Lock()
	delete(a.breakerCache, channelID)
	a.breakerMu.Unlock()
}
