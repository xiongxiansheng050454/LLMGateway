// Package storefake provides a test-only in-process implementation of the
// persistence ports. Production code must use store/postgres.
package storefake

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"LLMGateway/server/internal/accounts"
	"LLMGateway/server/internal/catalog"
	"LLMGateway/server/internal/money"
	"LLMGateway/server/internal/quota"
	"LLMGateway/server/internal/ratelimit"
	"LLMGateway/server/internal/store"
	"LLMGateway/server/internal/usage"
)

// Store is the in-process test implementation of the module-owned ports.
type Store struct {
	mu            sync.Mutex
	nextChannelID int
	nextModelID   int
	nextPricingID int
	channels      map[int]*catalog.Channel
	models        map[int]map[int]*catalog.ChannelModel
	pricing       map[string]*catalog.Pricing

	nextUserID   int
	nextKeyID    int
	nextTxID     int
	users        map[int]*accounts.User
	transactions map[int][]accounts.BalanceTransaction
	keys         map[int]*memoryKey
	orders       map[string]accounts.BalanceTransaction

	nextRateLimitID int
	rateLimits      map[int]*ratelimit.RateLimitRule
	nextUsageLogID  int
	usageLogs       []usage.UsageLog

	channelHealth              map[int]*catalog.ChannelHealth
	nextQuotaPolicyID          int
	nextQuotaReservationID     int64
	quotaPolicies              map[int]*quota.QuotaPolicy
	quotaBuckets               map[string]*fakeQuotaBucket
	quotaReservations          map[int64]*fakeQuotaReservation
	nextRateLimitReservationID int64
	rateLimitReservations      map[int64]ratelimit.RateLimitReservationInput
	breaker                    catalog.ChannelBreakerConfig
	now                        func() time.Time
	probes                     map[int]time.Time
}

var (
	_ accounts.Port      = (*Store)(nil)
	_ catalog.Port       = (*Store)(nil)
	_ catalog.HealthPort = (*Store)(nil)
	_ ratelimit.Port     = (*Store)(nil)
	_ usage.Port         = (*Store)(nil)
	_ quota.Port         = (*Store)(nil)
)

// memoryKey is the fake gateway key record. Only the hash is retained;
// the plaintext is returned once at creation/reset and never stored.
type memoryKey struct {
	id                 int
	userID             int
	keyName            string
	prefix             string
	keyHash            string
	permissions        json.RawMessage
	rateLimitOverrides json.RawMessage
	isActive           bool
	lastUsedAt         *string
	expiresAt          *string
}

// NewWithClock builds a fake store with an injected clock, used by tests to
// make cooldown behaviour deterministic.
func NewWithClock(now func() time.Time) *Store {
	s := New()
	if now != nil {
		s.now = now
	}
	return s
}

func New() *Store {
	return &Store{
		nextChannelID:              1,
		nextModelID:                1,
		nextPricingID:              1,
		channels:                   map[int]*catalog.Channel{},
		models:                     map[int]map[int]*catalog.ChannelModel{},
		pricing:                    map[string]*catalog.Pricing{},
		nextUserID:                 1,
		nextKeyID:                  1,
		nextTxID:                   1,
		users:                      map[int]*accounts.User{},
		transactions:               map[int][]accounts.BalanceTransaction{},
		keys:                       map[int]*memoryKey{},
		orders:                     map[string]accounts.BalanceTransaction{},
		nextRateLimitID:            1,
		rateLimits:                 map[int]*ratelimit.RateLimitRule{},
		nextUsageLogID:             1,
		nextQuotaPolicyID:          1,
		nextQuotaReservationID:     1,
		nextRateLimitReservationID: 1,
		rateLimitReservations:      map[int64]ratelimit.RateLimitReservationInput{},
		quotaPolicies:              map[int]*quota.QuotaPolicy{},
		quotaBuckets:               map[string]*fakeQuotaBucket{},
		quotaReservations:          map[int64]*fakeQuotaReservation{},
		channelHealth:              map[int]*catalog.ChannelHealth{},
		breaker:                    catalog.DefaultChannelBreakerConfig(),
		now:                        time.Now,
		probes:                     map[int]time.Time{},
	}
}

func (s *Store) channelDTO(ch *catalog.Channel) catalog.ChannelDTO {
	return catalog.ChannelDTO{ID: ch.ID, Name: ch.Name, BaseURL: ch.BaseURL, AuthType: ch.AuthType, Status: ch.Status, Weight: ch.Weight, Priority: ch.Priority, Balance: ch.Balance, ModelCount: len(s.models[ch.ID])}
}

func (s *Store) pricingDTO(p *catalog.Pricing) catalog.PricingDTO {
	channelName, upstream := "", ""
	if ch := s.channels[p.ChannelID]; ch != nil {
		channelName = ch.Name
	}
	for _, m := range s.models[p.ChannelID] {
		if m.ModelName == p.ModelName {
			upstream = m.UpstreamModel
			break
		}
	}
	return catalog.PricingDTO{ID: p.ID, ChannelID: p.ChannelID, ChannelName: channelName, ModelName: p.ModelName, UpstreamModel: upstream, InputPricePer1M: p.InputPricePer1M, OutputPricePer1M: p.OutputPricePer1M, CachedInputPricePer1M: p.CachedInputPricePer1M, Currency: p.Currency}
}

func (s *Store) hasChannelModelLocked(channelID int, modelName string) bool {
	for _, m := range s.models[channelID] {
		if m.ModelName == modelName {
			return true
		}
	}
	return false
}

// normalizeBalance canonicalizes an optional balance to 6 decimals, matching
// the PostgreSQL NUMERIC(20,6) column. An empty value means "unlimited" (nil).
func normalizeBalance(balance *string) (*string, error) {
	if balance == nil || *balance == "" {
		return nil, nil
	}
	amount, err := money.Parse6(*balance)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid balance", store.ErrInvalid)
	}
	formatted := money.Format6(amount)
	return &formatted, nil
}

func normalizePrice8(value string, field string) (string, error) {
	amount, err := money.Parse8(value)
	if err != nil {
		return "", fmt.Errorf("%w: invalid %s", store.ErrInvalid, field)
	}
	return money.Format8(amount), nil
}

func normalizeOptionalPrice8(value string, field string) (string, error) {
	if value == "" {
		return "", nil
	}
	return normalizePrice8(value, field)
}

func pricingKey(channelID int, model string) string {
	return fmt.Sprintf("%d:%s", channelID, model)
}
