package catalog

// Port is the catalog persistence primitive surface. It exposes only CRUD and
// query primitives: business rules, defaults, encryption and orchestration live
// on Server.
type Port interface {
	ListChannels() (ListResponse[ChannelDTO], error)
	GetChannelDTO(id int) (ChannelDTO, error)
	GetChannelRecord(id int) (ChannelRecord, error)
	InsertChannel(in ChannelInsert) (int, error)
	UpdateChannelRecord(id int, in ChannelUpdate) (bool, error)
	UpdateChannelStatusRecord(id, status int) (bool, error)
	DeleteChannel(id int) (bool, error)

	ListChannelModels(channelID int) (ListResponse[ChannelModel], error)
	InsertChannelModel(channelID int, in ChannelModel) (ChannelModel, error)
	UpdateChannelModelRecord(channelID, modelID int, upstreamModel string, enabled bool) (ChannelModel, bool, error)
	DeleteChannelModel(channelID, modelID int) (bool, error)
	// ChannelModelExists reports whether a model mapping exists on a channel.
	ChannelModelExists(channelID int, modelName string) (bool, error)

	ListCatalogModels(enabledOnly bool) (ListResponse[CatalogModelDTO], error)

	ListPricing() (ListResponse[PricingDTO], error)
	UpsertPricingRecord(in PricingRecord) (PricingDTO, error)
	DeletePricing(in DeletePricingInput) error
	GetPricing(channelID int, modelName string) (PricingDTO, error)

	// RouteCandidates returns enabled mappings on enabled, non-open channels for
	// a public model, ordered by priority desc, weight desc, channel id. The
	// caller supplies the breaker cooldown in seconds.
	RouteCandidates(modelName string, cooldownSeconds int) (ListResponse[RouteCandidate], error)
}
