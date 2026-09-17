package proxy

import (
	"errors"
	"net/http"
	"time"

	"LLMGateway/server/internal/store"
)

var (
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrInvalidRequest       = errors.New("invalid request")
	ErrStreamingUnsupported = errors.New("streaming is not supported")
	ErrRateLimited          = errors.New("rate limit exceeded")
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrNoHealthyChannel     = errors.New("no healthy channel available")
	ErrUpstream             = errors.New("upstream error")
)

type Service struct {
	store    store.Store
	client   *http.Client
	randIntN func(int) int
	now      func() time.Time
}

func NewService(st store.Store, client *http.Client, randIntN func(int) int, now func() time.Time) *Service {
	return &Service{store: st, client: client, randIntN: randIntN, now: now}
}
