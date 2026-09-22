package catalog

import (
	"fmt"
	"strings"

	"LLMGateway/server/internal/money"
)

func (a *Server) ListChannelModels(channelID int) (ListResponse[ChannelModel], error) {
	return a.store.ListChannelModels(channelID)
}

func (a *Server) CreateChannelModel(channelID int, in ChannelModel) (ChannelModel, error) {
	if _, err := a.store.GetChannelDTO(channelID); err != nil {
		return ChannelModel{}, err
	}
	return a.store.InsertChannelModel(channelID, in)
}

func (a *Server) UpdateChannelModel(channelID, modelID int, upstreamModel string, enabled bool) (ChannelModel, error) {
	model, ok, err := a.store.UpdateChannelModelRecord(channelID, modelID, upstreamModel, enabled)
	if err != nil {
		return ChannelModel{}, err
	}
	if !ok {
		return ChannelModel{}, ErrNotFound
	}
	return model, nil
}

func (a *Server) DeleteChannelModel(channelID, modelID int) error {
	ok, err := a.store.DeleteChannelModel(channelID, modelID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (a *Server) ListCatalogModels(enabledOnly bool) (ListResponse[CatalogModelDTO], error) {
	return a.store.ListCatalogModels(enabledOnly)
}

func (a *Server) ListPricing() (ListResponse[PricingDTO], error) {
	return a.store.ListPricing()
}

// UpsertPricing validates the target channel and model mapping, normalizes the
// 8-decimal prices and persists the row.
func (a *Server) UpsertPricing(in PricingInput) (PricingDTO, error) {
	if in.ChannelID <= 0 || strings.TrimSpace(in.ModelName) == "" {
		return PricingDTO{}, fmt.Errorf("%w: channel_id and model_name are required", ErrInvalid)
	}
	if _, err := a.store.GetChannelDTO(in.ChannelID); err != nil {
		return PricingDTO{}, err
	}
	exists, err := a.store.ChannelModelExists(in.ChannelID, in.ModelName)
	if err != nil {
		return PricingDTO{}, err
	}
	if !exists {
		return PricingDTO{}, fmt.Errorf("%w: model mapping not found", ErrInvalid)
	}

	inputPrice, err := normalizePrice(in.InputPricePer1M, "input_price_per_1m")
	if err != nil {
		return PricingDTO{}, err
	}
	outputPrice, err := normalizePrice(in.OutputPricePer1M, "output_price_per_1m")
	if err != nil {
		return PricingDTO{}, err
	}
	cachedPrice, err := normalizeOptionalPrice(in.CachedInputPricePer1M, "cached_input_price_per_1m")
	if err != nil {
		return PricingDTO{}, err
	}
	currency := in.Currency
	if currency == "" {
		currency = "USD"
	}

	return a.store.UpsertPricingRecord(PricingRecord{
		ChannelID:             in.ChannelID,
		ModelName:             in.ModelName,
		InputPricePer1M:       inputPrice,
		OutputPricePer1M:      outputPrice,
		CachedInputPricePer1M: cachedPrice,
		Currency:              currency,
	})
}

func (a *Server) DeletePricing(in DeletePricingInput) error {
	return a.store.DeletePricing(in)
}

func (a *Server) GetPricing(channelID int, modelName string) (PricingDTO, error) {
	return a.store.GetPricing(channelID, modelName)
}

func (a *Server) RouteCandidates(modelName string) (ListResponse[RouteCandidate], error) {
	return a.store.RouteCandidates(modelName, int(a.breaker.Cooldown.Seconds()))
}

func normalizePrice(value string, field string) (string, error) {
	amount, err := money.Parse8(value)
	if err != nil {
		return "", fmt.Errorf("%w: invalid %s", ErrInvalid, field)
	}
	return money.Format8(amount), nil
}

func normalizeOptionalPrice(value string, field string) (string, error) {
	if value == "" {
		return "", nil
	}
	return normalizePrice(value, field)
}
