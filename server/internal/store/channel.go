package store

import "LLMGateway/server/internal/domain"

// ChannelStore covers upstream channels, their model mappings and pricing.
type ChannelStore interface {
	ListChannels() (domain.ListResponse[domain.ChannelDTO], error)
	CreateChannel(domain.ChannelInput) (domain.ChannelDTO, error)
	UpdateChannel(int, domain.ChannelInput) (domain.ChannelDTO, error)
	UpdateChannelStatus(int, int) (domain.ChannelDTO, error)
	UpdateChannelBalance(int, string, string) (domain.ChannelDTO, error)
	DeleteChannel(int) error
	GetChannelSecret(int) (*domain.Channel, error)
	ListChannelModels(int) (domain.ListResponse[domain.ChannelModel], error)
	CreateChannelModel(int, domain.ChannelModel) (domain.ChannelModel, error)
	UpdateChannelModel(int, int, string, bool) (domain.ChannelModel, error)
	DeleteChannelModel(int, int) error
	ListCatalogModels(bool) (domain.ListResponse[domain.CatalogModelDTO], error)
	ListPricing() (domain.ListResponse[domain.PricingDTO], error)
	UpsertPricing(domain.PricingInput) (domain.PricingDTO, error)
	DeletePricing(domain.DeletePricingInput) error
	TestChannel(int) (domain.ChannelTestResultDTO, error)

	// GetPricing returns the single pricing row for a channel + public model.
	// Missing rows return ErrNotFound.
	GetPricing(channelID int, modelName string) (domain.PricingDTO, error)
	// RouteCandidates returns enabled model mappings on enabled channels for a
	// public model name, ordered by priority desc, weight desc, channel id.
	RouteCandidates(modelName string) (domain.ListResponse[domain.RouteCandidate], error)
}
