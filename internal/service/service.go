// Package service orchestrates the downstream proxy: authentication, channel
// routing, upstream forwarding, billing, rate limiting and usage logging.
//
// Dependency direction is handler -> service -> store; this package must never
// import internal/handler.
package service

import (
	"errors"
	"math/rand"
	"net/http"
	"time"

	"LLMGateway/internal/store"
)

// Service-level errors. The HTTP handler maps them to status codes and
// OpenAI-compatible error bodies.
var (
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrInvalidRequest       = errors.New("invalid request")
	ErrStreamingUnsupported = errors.New("streaming is not supported")
	ErrRateLimited          = errors.New("rate limit exceeded")
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrNoChannel            = errors.New("no available channel")
	ErrUpstream             = errors.New("upstream error")
)

// Proxy performs the downstream request lifecycle.
type Proxy struct {
	store    store.Store
	client   *http.Client
	randIntN func(int) int
	now      func() time.Time
}

type Option func(*Proxy)

// WithRandSource injects the random source used for weighted routing so tests
// can make selection deterministic.
func WithRandSource(fn func(int) int) Option {
	return func(p *Proxy) {
		if fn != nil {
			p.randIntN = fn
		}
	}
}

// WithClock injects the clock used for expiry and rate limit windows.
func WithClock(fn func() time.Time) Option {
	return func(p *Proxy) {
		if fn != nil {
			p.now = fn
		}
	}
}

func New(st store.Store, client *http.Client, opts ...Option) *Proxy {
	p := &Proxy{
		store:    st,
		client:   client,
		randIntN: rand.Intn,
		now:      time.Now,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}
