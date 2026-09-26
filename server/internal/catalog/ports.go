package catalog

import "context"

// Port is the catalog persistence primitive surface. It exposes only CRUD and
// query primitives: business rules, defaults, encryption and orchestration live
// on Server.
type Port interface {
	ListChannels(ctx context.Context) (ListResponse[ChannelDTO], error)
	GetChannelDTO(ctx context.Context, id int) (ChannelDTO, error)
	GetChannelRecord(ctx context.Context, id int) (ChannelRecord, error)
	InsertChannel(ctx context.Context, in ChannelInsert) (int, error)
	UpdateChannelRecord(ctx context.Context, id int, in ChannelUpdate) (bool, error)
	UpdateChannelStatusRecord(ctx context.Context, id, status int) (bool, error)
	DeleteChannel(ctx context.Context, id int) (bool, error)

	ListChannelModels(ctx context.Context, channelID int) (ListResponse[ChannelModel], error)
	InsertChannelModel(ctx context.Context, channelID int, in ChannelModel) (ChannelModel, error)
	UpdateChannelModelRecord(ctx context.Context, channelID, modelID int, upstreamModel string, enabled bool) (ChannelModel, bool, error)
	DeleteChannelModel(ctx context.Context, channelID, modelID int) (bool, error)
	// ChannelModelExists reports whether a model mapping exists on a channel.
	ChannelModelExists(ctx context.Context, channelID int, modelName string) (bool, error)

	ListCatalogModels(ctx context.Context, enabledOnly bool) (ListResponse[CatalogModelDTO], error)

	ListPricing(ctx context.Context) (ListResponse[PricingDTO], error)
	UpsertPricingRecord(ctx context.Context, in PricingRecord) (PricingDTO, error)
	DeletePricing(ctx context.Context, in DeletePricingInput) error
	GetPricing(ctx context.Context, channelID int, modelName string) (PricingDTO, error)

	// RouteCandidates returns enabled mappings on enabled, non-open channels for
	// a public model, ordered by priority desc, weight desc, channel id. The
	// caller supplies the breaker cooldown in seconds.
	RouteCandidates(ctx context.Context, modelName string, cooldownSeconds int) (ListResponse[RouteCandidate], error)
}
