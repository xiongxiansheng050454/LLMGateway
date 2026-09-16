package store

import "LLMGateway/server/internal/domain"

// ChannelStore covers upstream channels, their model mappings and pricing.
type ChannelStore interface {
	ListChannels() (domain.ListResponse, error)
	CreateChannel(domain.ChannelInput) (map[string]any, error)
	UpdateChannel(int, domain.ChannelInput) (map[string]any, error)
	UpdateChannelStatus(int, int) (map[string]any, error)
	UpdateChannelBalance(int, string, string) (map[string]any, error)
	DeleteChannel(int) error
	GetChannelSecret(int) (*domain.Channel, error)
	ListChannelModels(int) (domain.ListResponse, error)
	CreateChannelModel(int, domain.ChannelModel) (domain.ChannelModel, error)
	UpdateChannelModel(int, int, string, bool) (domain.ChannelModel, error)
	DeleteChannelModel(int, int) error
	ListCatalogModels(bool) (domain.ListResponse, error)
	ListPricing() (domain.ListResponse, error)
	UpsertPricing(domain.PricingInput) (map[string]any, error)
	DeletePricing(domain.DeletePricingInput) error
	TestChannel(int) (map[string]any, error)

	// GetPricing returns the single pricing row for a channel + public model.
	// Missing rows return ErrNotFound.
	GetPricing(channelID int, modelName string) (map[string]any, error)
	// RouteCandidates returns enabled model mappings on enabled channels for a
	// public model name, ordered by priority desc, weight desc, channel id.
	RouteCandidates(modelName string) (domain.ListResponse, error)
}
