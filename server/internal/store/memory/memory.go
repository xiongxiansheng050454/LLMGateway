package memory

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/money"
	"LLMGateway/server/internal/store"
)

// Store is the in-memory implementation of store.Store. It is the default for
// local runs, tests and feature work before PostgreSQL is wired up.
type Store struct {
	mu            sync.Mutex
	nextChannelID int
	nextModelID   int
	nextPricingID int
	channels      map[int]*domain.Channel
	models        map[int]map[int]*domain.ChannelModel
	pricing       map[string]*domain.Pricing

	nextUserID   int
	nextKeyID    int
	nextTxID     int
	users        map[int]*domain.User
	transactions map[int][]domain.BalanceTransaction
	keys         map[int]*memoryKey
	orders       map[string]domain.BalanceTransaction

	nextRateLimitID int
	rateLimits      map[int]*domain.RateLimitRule
	nextUsageLogID  int
	usageLogs       []domain.UsageLog

	channelHealth map[int]*domain.ChannelHealth
	breaker       domain.ChannelBreakerConfig
	now           func() time.Time
}

var (
	_ store.Store              = (*Store)(nil)
	_ store.ChannelStore       = (*Store)(nil)
	_ store.UserStore          = (*Store)(nil)
	_ store.RateLimitStore     = (*Store)(nil)
	_ store.UsageStore         = (*Store)(nil)
	_ store.ChannelHealthStore = (*Store)(nil)
)

// memoryKey is the in-memory gateway key record. Only the hash is retained;
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

// NewWithClock builds a memory store with an injected clock, used by tests to
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
		nextChannelID:   1,
		nextModelID:     1,
		nextPricingID:   1,
		channels:        map[int]*domain.Channel{},
		models:          map[int]map[int]*domain.ChannelModel{},
		pricing:         map[string]*domain.Pricing{},
		nextUserID:      1,
		nextKeyID:       1,
		nextTxID:        1,
		users:           map[int]*domain.User{},
		transactions:    map[int][]domain.BalanceTransaction{},
		keys:            map[int]*memoryKey{},
		orders:          map[string]domain.BalanceTransaction{},
		nextRateLimitID: 1,
		rateLimits:      map[int]*domain.RateLimitRule{},
		nextUsageLogID:  1,
		channelHealth:   map[int]*domain.ChannelHealth{},
		breaker:         domain.DefaultChannelBreakerConfig(),
		now:             time.Now,
	}
}

func (s *Store) channelDTO(ch *domain.Channel) domain.ChannelDTO {
	return domain.ChannelDTO{ID: ch.ID, Name: ch.Name, BaseURL: ch.BaseURL, AuthType: ch.AuthType, Status: ch.Status, Weight: ch.Weight, Priority: ch.Priority, Balance: ch.Balance, ModelCount: len(s.models[ch.ID])}
}

func (s *Store) pricingDTO(p *domain.Pricing) domain.PricingDTO {
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
	return domain.PricingDTO{ID: p.ID, ChannelID: p.ChannelID, ChannelName: channelName, ModelName: p.ModelName, UpstreamModel: upstream, InputPricePer1M: p.InputPricePer1M, OutputPricePer1M: p.OutputPricePer1M, CachedInputPricePer1M: p.CachedInputPricePer1M, Currency: p.Currency}
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
