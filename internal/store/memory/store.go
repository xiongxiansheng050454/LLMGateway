package memory

import (
	"fmt"
	"strings"
	"sync"

	"LLMGateway/internal/domain"
	"LLMGateway/internal/money"
	"LLMGateway/internal/store"
)

type Store struct {
	mu            sync.Mutex
	nextChannelID int
	nextModelID   int
	nextPricingID int
	channels      map[int]*domain.Channel
	models        map[int]map[int]*domain.ChannelModel
	pricing       map[string]*domain.Pricing
}

func New() *Store {
	return &Store{
		nextChannelID: 1,
		nextModelID:   1,
		nextPricingID: 1,
		channels:      map[int]*domain.Channel{},
		models:        map[int]map[int]*domain.ChannelModel{},
		pricing:       map[string]*domain.Pricing{},
	}
}

func (s *Store) ListChannels() (domain.ListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := []any{}
	for _, ch := range s.channels {
		list = append(list, s.channelDTO(ch))
	}
	return domain.ListResponse{List: list, Total: len(list)}, nil
}

func (s *Store) CreateChannel(in domain.ChannelInput) (map[string]any, error) {
	if strings.TrimSpace(in.APIKey) == "" {
		return nil, fmt.Errorf("%w: api_key is required", store.ErrInvalid)
	}
	if in.AuthType == "" {
		in.AuthType = "bearer"
	}
	if in.Weight == 0 {
		in.Weight = 100
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	ch := &domain.Channel{ID: s.nextChannelID, Name: in.Name, BaseURL: in.BaseURL, APIKey: in.APIKey, AuthType: in.AuthType, Status: in.Status, Weight: in.Weight, Priority: in.Priority, Balance: cleanBalance(in.Balance)}
	s.nextChannelID++
	s.channels[ch.ID] = ch
	return s.channelDTO(ch), nil
}

func (s *Store) UpdateChannel(id int, in domain.ChannelInput) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.channels[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	ch.Name, ch.BaseURL, ch.AuthType, ch.Status, ch.Weight, ch.Priority = in.Name, in.BaseURL, in.AuthType, in.Status, in.Weight, in.Priority
	if strings.TrimSpace(in.APIKey) != "" {
		ch.APIKey = in.APIKey
	}
	ch.Balance = cleanBalance(in.Balance)
	return s.channelDTO(ch), nil
}

func (s *Store) UpdateChannelStatus(id int, status int) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.channels[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	ch.Status = status
	return s.channelDTO(ch), nil
}

func (s *Store) UpdateChannelBalance(id int, balance string, delta string) (map[string]any, error) {
	if balance == "" && delta == "" {
		return nil, fmt.Errorf("%w: balance or delta is required", store.ErrInvalid)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.channels[id]
	if !ok {
		return nil, store.ErrNotFound
	}

	base := money.Amount(0)
	if balance != "" {
		parsed, err := money.Parse6(balance)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid balance", store.ErrInvalid)
		}
		base = parsed
	} else if ch.Balance != nil {
		parsed, err := money.Parse6(*ch.Balance)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid balance", store.ErrInvalid)
		}
		base = parsed
	}
	if delta != "" {
		parsed, err := money.Parse6(delta)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid delta", store.ErrInvalid)
		}
		base = base.Add(parsed)
	}
	formatted := money.Format6(base)
	ch.Balance = &formatted
	return s.channelDTO(ch), nil
}

func (s *Store) DeleteChannel(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.channels[id]; !ok {
		return store.ErrNotFound
	}
	delete(s.channels, id)
	delete(s.models, id)
	for key, p := range s.pricing {
		if p.ChannelID == id {
			delete(s.pricing, key)
		}
	}
	return nil
}

func (s *Store) GetChannelSecret(id int) (*domain.Channel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.channels[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	copy := *ch
	return &copy, nil
}

func (s *Store) ListChannelModels(channelID int) (domain.ListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := []any{}
	for _, m := range s.models[channelID] {
		list = append(list, *m)
	}
	return domain.ListResponse{List: list, Total: len(list)}, nil
}

func (s *Store) CreateChannelModel(channelID int, in domain.ChannelModel) (domain.ChannelModel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.channels[channelID]; !ok {
		return domain.ChannelModel{}, store.ErrNotFound
	}
	in.ID = s.nextModelID
	s.nextModelID++
	if s.models[channelID] == nil {
		s.models[channelID] = map[int]*domain.ChannelModel{}
	}
	m := in
	s.models[channelID][m.ID] = &m
	return m, nil
}

func (s *Store) UpdateChannelModel(channelID, modelID int, upstreamModel string, enabled bool) (domain.ChannelModel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.models[channelID][modelID]
	if !ok {
		return domain.ChannelModel{}, store.ErrNotFound
	}
	m.UpstreamModel, m.Enabled = upstreamModel, enabled
	return *m, nil
}

func (s *Store) DeleteChannelModel(channelID, modelID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.channels[channelID]; !ok {
		return store.ErrNotFound
	}
	if _, ok := s.models[channelID][modelID]; !ok {
		return store.ErrNotFound
	}
	delete(s.models[channelID], modelID)
	return nil
}

func (s *Store) ListCatalogModels(enabledOnly bool) (domain.ListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	byName := map[string]map[string]any{}
	for channelID, models := range s.models {
		ch := s.channels[channelID]
		if ch == nil {
			continue
		}
		for _, m := range models {
			if enabledOnly && !m.Enabled {
				continue
			}
			entry := byName[m.ModelName]
			if entry == nil {
				entry = map[string]any{"model_name": m.ModelName, "status": 1, "channels": []any{}}
				byName[m.ModelName] = entry
			}
			entry["channels"] = append(entry["channels"].([]any), map[string]any{"channel_id": channelID, "channel_name": ch.Name, "upstream_model": m.UpstreamModel, "enabled": m.Enabled})
		}
	}
	list := []any{}
	for _, item := range byName {
		list = append(list, item)
	}
	return domain.ListResponse{List: list, Total: len(list)}, nil
}

func (s *Store) ListPricing() (domain.ListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := []any{}
	for _, p := range s.pricing {
		list = append(list, s.pricingDTO(p))
	}
	return domain.ListResponse{List: list, Total: len(list)}, nil
}

func (s *Store) UpsertPricing(in domain.PricingInput) (map[string]any, error) {
	if in.ChannelID <= 0 || strings.TrimSpace(in.ModelName) == "" {
		return nil, fmt.Errorf("%w: channel_id and model_name are required", store.ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.channels[in.ChannelID]; !ok {
		return nil, store.ErrNotFound
	}
	if !s.hasChannelModelLocked(in.ChannelID, in.ModelName) {
		return nil, fmt.Errorf("%w: model mapping not found", store.ErrInvalid)
	}

	key := pricingKey(in.ChannelID, in.ModelName)
	p := s.pricing[key]
	if p == nil {
		p = &domain.Pricing{ID: s.nextPricingID, ChannelID: in.ChannelID, ModelName: in.ModelName}
		s.nextPricingID++
		s.pricing[key] = p
	}
	p.InputPricePer1M, p.OutputPricePer1M, p.CachedInputPricePer1M, p.Currency = in.InputPricePer1M, in.OutputPricePer1M, in.CachedInputPricePer1M, in.Currency
	if p.Currency == "" {
		p.Currency = "USD"
	}
	return s.pricingDTO(p), nil
}

func (s *Store) DeletePricing(in domain.DeletePricingInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pricing, pricingKey(in.ChannelID, in.ModelName))
	return nil
}

func (s *Store) TestChannel(channelID int) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.channels[channelID]; !ok {
		return nil, store.ErrNotFound
	}
	items := []any{}
	for _, m := range s.models[channelID] {
		items = append(items, map[string]any{"model_alias": m.ModelName, "upstream_model": m.UpstreamModel, "http_status": 0, "latency_ms": 0, "ok": false, "error": "not tested in MVP"})
	}
	return map[string]any{"list": items}, nil
}

func (s *Store) channelDTO(ch *domain.Channel) map[string]any {
	return map[string]any{"id": ch.ID, "name": ch.Name, "base_url": ch.BaseURL, "auth_type": ch.AuthType, "status": ch.Status, "weight": ch.Weight, "priority": ch.Priority, "balance": ch.Balance, "model_count": len(s.models[ch.ID])}
}

func (s *Store) pricingDTO(p *domain.Pricing) map[string]any {
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
	return map[string]any{"id": p.ID, "channel_id": p.ChannelID, "channel_name": channelName, "model_name": p.ModelName, "upstream_model": upstream, "input_price_per_1m": p.InputPricePer1M, "output_price_per_1m": p.OutputPricePer1M, "cached_input_price_per_1m": p.CachedInputPricePer1M, "currency": p.Currency}
}

func (s *Store) hasChannelModelLocked(channelID int, modelName string) bool {
	for _, m := range s.models[channelID] {
		if m.ModelName == modelName {
			return true
		}
	}
	return false
}

func cleanBalance(balance *string) *string {
	if balance == nil || *balance == "" {
		return nil
	}
	return balance
}

func pricingKey(channelID int, model string) string {
	return fmt.Sprintf("%d:%s", channelID, model)
}
