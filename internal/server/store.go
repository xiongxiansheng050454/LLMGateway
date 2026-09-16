package server

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

var (
	errNotFound = errors.New("not found")
	errInvalid  = errors.New("invalid")
)

type Store interface {
	ListChannels() listResponse
	CreateChannel(channelInput) (map[string]any, error)
	UpdateChannel(int, channelInput) (map[string]any, error)
	UpdateChannelStatus(int, int) (map[string]any, error)
	UpdateChannelBalance(int, string, string) (map[string]any, error)
	DeleteChannel(int) error
	GetChannelSecret(int) (*channel, error)
	ListChannelModels(int) listResponse
	CreateChannelModel(int, channelModel) (channelModel, error)
	UpdateChannelModel(int, int, string, bool) (channelModel, error)
	DeleteChannelModel(int, int) error
	ListCatalogModels(bool) listResponse
	ListPricing() listResponse
	UpsertPricing(pricingInput) (map[string]any, error)
	DeletePricing(deletePricingInput) error
	TestChannel(int) (map[string]any, error)
}

type memoryStore struct {
	mu            sync.Mutex
	nextChannelID int
	nextModelID   int
	nextPricingID int
	channels      map[int]*channel
	models        map[int]map[int]*channelModel
	pricing       map[string]*pricing
}

func NewMemoryStore() Store {
	return &memoryStore{
		nextChannelID: 1,
		nextModelID:   1,
		nextPricingID: 1,
		channels:      map[int]*channel{},
		models:        map[int]map[int]*channelModel{},
		pricing:       map[string]*pricing{},
	}
}

func (s *memoryStore) ListChannels() listResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := []any{}
	for _, ch := range s.channels {
		list = append(list, s.channelDTO(ch))
	}
	return listResponse{List: list, Total: len(list)}
}

func (s *memoryStore) CreateChannel(in channelInput) (map[string]any, error) {
	if strings.TrimSpace(in.APIKey) == "" {
		return nil, fmt.Errorf("%w: api_key is required", errInvalid)
	}
	if in.AuthType == "" {
		in.AuthType = "bearer"
	}
	if in.Weight == 0 {
		in.Weight = 100
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	ch := &channel{ID: s.nextChannelID, Name: in.Name, BaseURL: in.BaseURL, APIKey: in.APIKey, AuthType: in.AuthType, Status: in.Status, Weight: in.Weight, Priority: in.Priority, Balance: cleanBalance(in.Balance)}
	s.nextChannelID++
	s.channels[ch.ID] = ch
	return s.channelDTO(ch), nil
}

func (s *memoryStore) UpdateChannel(id int, in channelInput) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.channels[id]
	if !ok {
		return nil, errNotFound
	}
	ch.Name, ch.BaseURL, ch.AuthType, ch.Status, ch.Weight, ch.Priority = in.Name, in.BaseURL, in.AuthType, in.Status, in.Weight, in.Priority
	if strings.TrimSpace(in.APIKey) != "" {
		ch.APIKey = in.APIKey
	}
	ch.Balance = cleanBalance(in.Balance)
	return s.channelDTO(ch), nil
}

func (s *memoryStore) UpdateChannelStatus(id int, status int) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.channels[id]
	if !ok {
		return nil, errNotFound
	}
	ch.Status = status
	return s.channelDTO(ch), nil
}

func (s *memoryStore) UpdateChannelBalance(id int, balance string, delta string) (map[string]any, error) {
	if balance == "" && delta == "" {
		return nil, fmt.Errorf("%w: balance or delta is required", errInvalid)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.channels[id]
	if !ok {
		return nil, errNotFound
	}

	base := int64(0)
	var err error
	if balance != "" {
		base, err = parseMoney6(balance)
	} else if ch.Balance != nil {
		base, err = parseMoney6(*ch.Balance)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: invalid balance", errInvalid)
	}
	if delta != "" {
		d, err := parseMoney6(delta)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid delta", errInvalid)
		}
		base += d
	}
	b := formatMoney6(base)
	ch.Balance = &b
	return s.channelDTO(ch), nil
}

func (s *memoryStore) DeleteChannel(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.channels[id]; !ok {
		return errNotFound
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

func (s *memoryStore) GetChannelSecret(id int) (*channel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.channels[id]
	if !ok {
		return nil, errNotFound
	}
	copy := *ch
	return &copy, nil
}

func (s *memoryStore) ListChannelModels(channelID int) listResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := []any{}
	for _, m := range s.models[channelID] {
		list = append(list, *m)
	}
	return listResponse{List: list, Total: len(list)}
}

func (s *memoryStore) CreateChannelModel(channelID int, in channelModel) (channelModel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.channels[channelID]; !ok {
		return channelModel{}, errNotFound
	}
	in.ID = s.nextModelID
	s.nextModelID++
	if s.models[channelID] == nil {
		s.models[channelID] = map[int]*channelModel{}
	}
	m := in
	s.models[channelID][m.ID] = &m
	return m, nil
}

func (s *memoryStore) UpdateChannelModel(channelID, modelID int, upstreamModel string, enabled bool) (channelModel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.models[channelID][modelID]
	if !ok {
		return channelModel{}, errNotFound
	}
	m.UpstreamModel, m.Enabled = upstreamModel, enabled
	return *m, nil
}

func (s *memoryStore) DeleteChannelModel(channelID, modelID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.channels[channelID]; !ok {
		return errNotFound
	}
	if _, ok := s.models[channelID][modelID]; !ok {
		return errNotFound
	}
	delete(s.models[channelID], modelID)
	return nil
}

func (s *memoryStore) ListCatalogModels(enabledOnly bool) listResponse {
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
	return listResponse{List: list, Total: len(list)}
}

func (s *memoryStore) ListPricing() listResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := []any{}
	for _, p := range s.pricing {
		list = append(list, s.pricingDTO(p))
	}
	return listResponse{List: list, Total: len(list)}
}

func (s *memoryStore) UpsertPricing(in pricingInput) (map[string]any, error) {
	if in.ChannelID <= 0 || strings.TrimSpace(in.ModelName) == "" {
		return nil, fmt.Errorf("%w: channel_id and model_name are required", errInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.channels[in.ChannelID]; !ok {
		return nil, errNotFound
	}
	if !s.hasChannelModelLocked(in.ChannelID, in.ModelName) {
		return nil, fmt.Errorf("%w: model mapping not found", errInvalid)
	}

	key := pricingKey(in.ChannelID, in.ModelName)
	p := s.pricing[key]
	if p == nil {
		p = &pricing{ID: s.nextPricingID, ChannelID: in.ChannelID, ModelName: in.ModelName}
		s.nextPricingID++
		s.pricing[key] = p
	}
	p.InputPricePer1M, p.OutputPricePer1M, p.CachedInputPricePer1M, p.Currency = in.InputPricePer1M, in.OutputPricePer1M, in.CachedInputPricePer1M, in.Currency
	if p.Currency == "" {
		p.Currency = "USD"
	}
	return s.pricingDTO(p), nil
}

func (s *memoryStore) DeletePricing(in deletePricingInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pricing, pricingKey(in.ChannelID, in.ModelName))
	return nil
}

func (s *memoryStore) TestChannel(channelID int) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.channels[channelID]; !ok {
		return nil, errNotFound
	}
	items := []any{}
	for _, m := range s.models[channelID] {
		items = append(items, map[string]any{"model_alias": m.ModelName, "upstream_model": m.UpstreamModel, "http_status": 0, "latency_ms": 0, "ok": false, "error": "not tested in MVP"})
	}
	return map[string]any{"list": items}, nil
}

func (s *memoryStore) channelDTO(ch *channel) map[string]any {
	return map[string]any{"id": ch.ID, "name": ch.Name, "base_url": ch.BaseURL, "auth_type": ch.AuthType, "status": ch.Status, "weight": ch.Weight, "priority": ch.Priority, "balance": ch.Balance, "model_count": len(s.models[ch.ID])}
}

func (s *memoryStore) pricingDTO(p *pricing) map[string]any {
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

func (s *memoryStore) hasChannelModelLocked(channelID int, modelName string) bool {
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

func parseMoney6(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	negative := strings.HasPrefix(value, "-")
	if negative {
		value = strings.TrimPrefix(value, "-")
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, errInvalid
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, errInvalid
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > 6 {
		return 0, errInvalid
	}
	for len(fraction) < 6 {
		fraction += "0"
	}
	frac, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, errInvalid
	}
	amount := whole*1_000_000 + frac
	if negative {
		return -amount, nil
	}
	return amount, nil
}

func formatMoney6(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	return fmt.Sprintf("%s%d.%06d", sign, value/1_000_000, value%1_000_000)
}
