package memory

import (
	"fmt"
	"sort"
	"strings"

	"LLMGateway/internal/domain"
	"LLMGateway/internal/money"
	"LLMGateway/internal/store"
)

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

	balance, err := normalizeBalance(in.Balance)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	ch := &domain.Channel{ID: s.nextChannelID, Name: in.Name, BaseURL: in.BaseURL, APIKey: in.APIKey, AuthType: in.AuthType, Status: in.Status, Weight: in.Weight, Priority: in.Priority, Balance: balance}
	s.nextChannelID++
	s.channels[ch.ID] = ch
	return s.channelDTO(ch), nil
}

func (s *Store) UpdateChannel(id int, in domain.ChannelInput) (map[string]any, error) {
	balance, err := normalizeBalance(in.Balance)
	if err != nil {
		return nil, err
	}

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
	ch.Balance = balance
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
	if s.hasChannelModelLocked(channelID, in.ModelName) {
		return domain.ChannelModel{}, fmt.Errorf("%w: model mapping already exists", store.ErrInvalid)
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

	inputPrice, err := normalizePrice8(in.InputPricePer1M, "input_price_per_1m")
	if err != nil {
		return nil, err
	}
	outputPrice, err := normalizePrice8(in.OutputPricePer1M, "output_price_per_1m")
	if err != nil {
		return nil, err
	}
	cachedPrice, err := normalizeOptionalPrice8(in.CachedInputPricePer1M, "cached_input_price_per_1m")
	if err != nil {
		return nil, err
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
	p.InputPricePer1M, p.OutputPricePer1M, p.CachedInputPricePer1M, p.Currency = inputPrice, outputPrice, cachedPrice, in.Currency
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

func (s *Store) GetPricing(channelID int, modelName string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pricing, ok := s.pricing[pricingKey(channelID, modelName)]
	if !ok {
		return nil, store.ErrNotFound
	}
	return s.pricingDTO(pricing), nil
}

func (s *Store) RouteCandidates(modelName string) (domain.ListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	candidates := []map[string]any{}
	for channelID, models := range s.models {
		channel := s.channels[channelID]
		if channel == nil || channel.Status != 1 {
			continue
		}
		for _, model := range models {
			if model.ModelName != modelName || !model.Enabled {
				continue
			}
			candidates = append(candidates, map[string]any{
				"channel_id":     channelID,
				"channel_name":   channel.Name,
				"upstream_model": model.UpstreamModel,
				"priority":       channel.Priority,
				"weight":         channel.Weight,
				"balance":        channel.Balance,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i]["priority"].(int) != candidates[j]["priority"].(int) {
			return candidates[i]["priority"].(int) > candidates[j]["priority"].(int)
		}
		if candidates[i]["weight"].(int) != candidates[j]["weight"].(int) {
			return candidates[i]["weight"].(int) > candidates[j]["weight"].(int)
		}
		return candidates[i]["channel_id"].(int) < candidates[j]["channel_id"].(int)
	})

	list := []any{}
	for _, candidate := range candidates {
		list = append(list, candidate)
	}
	return domain.ListResponse{List: list, Total: len(list)}, nil
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
