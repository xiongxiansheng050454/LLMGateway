package postgres

import (
	"LLMGateway/internal/domain"
	"LLMGateway/internal/store"
)

func (s *Store) ListChannels() (domain.ListResponse, error) {
	return domain.ListResponse{}, store.ErrNotImplemented
}

func (s *Store) CreateChannel(domain.ChannelInput) (map[string]any, error) {
	return nil, store.ErrNotImplemented
}

func (s *Store) UpdateChannel(int, domain.ChannelInput) (map[string]any, error) {
	return nil, store.ErrNotImplemented
}

func (s *Store) UpdateChannelStatus(int, int) (map[string]any, error) {
	return nil, store.ErrNotImplemented
}

func (s *Store) UpdateChannelBalance(int, string, string) (map[string]any, error) {
	return nil, store.ErrNotImplemented
}

func (s *Store) DeleteChannel(int) error {
	return store.ErrNotImplemented
}

func (s *Store) GetChannelSecret(int) (*domain.Channel, error) {
	return nil, store.ErrNotImplemented
}

func (s *Store) ListChannelModels(int) (domain.ListResponse, error) {
	return domain.ListResponse{}, store.ErrNotImplemented
}

func (s *Store) CreateChannelModel(int, domain.ChannelModel) (domain.ChannelModel, error) {
	return domain.ChannelModel{}, store.ErrNotImplemented
}

func (s *Store) UpdateChannelModel(int, int, string, bool) (domain.ChannelModel, error) {
	return domain.ChannelModel{}, store.ErrNotImplemented
}

func (s *Store) DeleteChannelModel(int, int) error {
	return store.ErrNotImplemented
}

func (s *Store) ListCatalogModels(bool) (domain.ListResponse, error) {
	return domain.ListResponse{}, store.ErrNotImplemented
}

func (s *Store) ListPricing() (domain.ListResponse, error) {
	return domain.ListResponse{}, store.ErrNotImplemented
}

func (s *Store) UpsertPricing(domain.PricingInput) (map[string]any, error) {
	return nil, store.ErrNotImplemented
}

func (s *Store) DeletePricing(domain.DeletePricingInput) error {
	return store.ErrNotImplemented
}

func (s *Store) TestChannel(int) (map[string]any, error) {
	return nil, store.ErrNotImplemented
}
