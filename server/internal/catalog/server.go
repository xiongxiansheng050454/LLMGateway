package catalog

import (
	"net/http"
	"time"

	"LLMGateway/server/internal/crypto"
)

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
		store:       d.Store,
		health:      d.Health,
		tx:          d.Tx,
		cipher:      d.Cipher,
		client:      d.Client,
		now:         now,
		breaker:     breaker,
		testTimeout: channelTestTimeout,
	}
}
