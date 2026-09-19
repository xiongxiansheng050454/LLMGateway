package catalog

// ChannelStore covers upstream channels, their model mappings and pricing.
type Port interface {
	ListChannels() (ListResponse[ChannelDTO], error)
	CreateChannel(ChannelInput) (ChannelDTO, error)
	UpdateChannel(int, ChannelInput) (ChannelDTO, error)
	UpdateChannelStatus(int, int) (ChannelDTO, error)
	UpdateChannelBalance(int, string, string) (ChannelDTO, error)
	DeleteChannel(int) error
	GetChannelSecret(int) (*Channel, error)
	ListChannelModels(int) (ListResponse[ChannelModel], error)
	CreateChannelModel(int, ChannelModel) (ChannelModel, error)
	UpdateChannelModel(int, int, string, bool) (ChannelModel, error)
	DeleteChannelModel(int, int) error
	ListCatalogModels(bool) (ListResponse[CatalogModelDTO], error)
	ListPricing() (ListResponse[PricingDTO], error)
	UpsertPricing(PricingInput) (PricingDTO, error)
	DeletePricing(DeletePricingInput) error
	// GetPricing returns the single pricing row for a channel + public model.
	// Missing rows return ErrNotFound.
	GetPricing(channelID int, modelName string) (PricingDTO, error)
	// RouteCandidates returns enabled model mappings on enabled channels for a
	// public model name, ordered by priority desc, weight desc, channel id.
	RouteCandidates(modelName string) (ListResponse[RouteCandidate], error)
}
