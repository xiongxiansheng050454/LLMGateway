package postgres

import (
	"context"
	"errors"

	domain "LLMGateway/server/internal/catalog"
	"LLMGateway/server/internal/db/sqlc"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListChannels() (domain.ListResponse[domain.ChannelDTO], error) {
	rows, err := s.queries.ListChannels(context.Background())
	if err != nil {
		return domain.ListResponse[domain.ChannelDTO]{}, mapError(err)
	}
	list := []domain.ChannelDTO{}
	for _, row := range rows {
		list = append(list, channelDTO(row.ID, row.Name, row.BaseUrl, row.AuthType, row.Status, row.Weight, row.Priority, textValue(row.Balance), row.ModelCount))
	}
	return domain.ListResponse[domain.ChannelDTO]{List: list, Total: len(list)}, nil
}

func (s *Store) GetChannelDTO(id int) (domain.ChannelDTO, error) {
	row, err := s.queries.GetChannel(context.Background(), int64(id))
	if err != nil {
		return domain.ChannelDTO{}, mapError(err)
	}
	return channelDTO(row.ID, row.Name, row.BaseUrl, row.AuthType, row.Status, row.Weight, row.Priority, textValue(row.Balance), row.ModelCount), nil
}

func (s *Store) GetChannelRecord(id int) (domain.ChannelRecord, error) {
	row, err := s.queries.GetChannelSecret(context.Background(), int64(id))
	if err != nil {
		return domain.ChannelRecord{}, mapError(err)
	}
	return domain.ChannelRecord{
		ID:               int(row.ID),
		Name:             row.Name,
		BaseURL:          row.BaseUrl,
		APIKeyCiphertext: row.ApiKeyCiphertext,
		AuthType:         row.AuthType,
		Status:           int(row.Status),
		Weight:           int(row.Weight),
		Priority:         int(row.Priority),
		Balance:          optionalString(textValue(row.Balance)),
	}, nil
}

func (s *Store) InsertChannel(in domain.ChannelInsert) (int, error) {
	var balance any
	if in.Balance != nil {
		balance = *in.Balance
	}
	id, err := s.queries.CreateChannel(context.Background(), sqlc.CreateChannelParams{
		Name:             in.Name,
		BaseUrl:          in.BaseURL,
		ApiKeyCiphertext: in.APIKeyCiphertext,
		AuthType:         in.AuthType,
		Status:           int32(in.Status),
		Weight:           int32(in.Weight),
		Priority:         int32(in.Priority),
		Balance:          balance,
	})
	if err != nil {
		return 0, mapError(err)
	}
	return int(id), nil
}

func (s *Store) UpdateChannelRecord(id int, in domain.ChannelUpdate) (bool, error) {
	var balance any
	if in.Balance != nil {
		balance = *in.Balance
	}
	affected, err := s.queries.UpdateChannel(context.Background(), sqlc.UpdateChannelParams{
		Name:             in.Name,
		BaseUrl:          in.BaseURL,
		AuthType:         in.AuthType,
		Status:           int32(in.Status),
		Weight:           int32(in.Weight),
		Priority:         int32(in.Priority),
		Balance:          balance,
		ApiKeyCiphertext: in.APIKeyCiphertext,
		ID:               int64(id),
	})
	if err != nil {
		return false, mapError(err)
	}
	return affected > 0, nil
}

func (s *Store) UpdateChannelStatusRecord(id, status int) (bool, error) {
	affected, err := s.queries.UpdateChannelStatus(context.Background(), sqlc.UpdateChannelStatusParams{Status: int32(status), ID: int64(id)})
	if err != nil {
		return false, mapError(err)
	}
	return affected > 0, nil
}

func (s *Store) DeleteChannel(id int) (bool, error) {
	affected, err := s.queries.DeleteChannel(context.Background(), int64(id))
	if err != nil {
		return false, mapError(err)
	}
	return affected > 0, nil
}

func (s *Store) ListChannelModels(channelID int) (domain.ListResponse[domain.ChannelModel], error) {
	rows, err := s.queries.ListChannelModels(context.Background(), int64(channelID))
	if err != nil {
		return domain.ListResponse[domain.ChannelModel]{}, mapError(err)
	}
	list := []domain.ChannelModel{}
	for _, row := range rows {
		list = append(list, domain.ChannelModel{ID: int(row.ID), ModelName: row.ModelName, UpstreamModel: row.UpstreamModel, Enabled: row.Enabled})
	}
	return domain.ListResponse[domain.ChannelModel]{List: list, Total: len(list)}, nil
}

func (s *Store) InsertChannelModel(channelID int, in domain.ChannelModel) (domain.ChannelModel, error) {
	row, err := s.queries.CreateChannelModel(context.Background(), sqlc.CreateChannelModelParams{
		ChannelID:     int64(channelID),
		ModelName:     in.ModelName,
		UpstreamModel: in.UpstreamModel,
		Enabled:       in.Enabled,
	})
	if err != nil {
		return domain.ChannelModel{}, mapError(err)
	}
	return domain.ChannelModel{ID: int(row.ID), ModelName: row.ModelName, UpstreamModel: row.UpstreamModel, Enabled: row.Enabled}, nil
}

func (s *Store) UpdateChannelModelRecord(channelID, modelID int, upstreamModel string, enabled bool) (domain.ChannelModel, bool, error) {
	row, err := s.queries.UpdateChannelModel(context.Background(), sqlc.UpdateChannelModelParams{
		UpstreamModel: upstreamModel,
		Enabled:       enabled,
		ChannelID:     int64(channelID),
		ID:            int64(modelID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ChannelModel{}, false, nil
		}
		return domain.ChannelModel{}, false, mapError(err)
	}
	return domain.ChannelModel{ID: int(row.ID), ModelName: row.ModelName, UpstreamModel: row.UpstreamModel, Enabled: row.Enabled}, true, nil
}

func (s *Store) DeleteChannelModel(channelID, modelID int) (bool, error) {
	affected, err := s.queries.DeleteChannelModel(context.Background(), sqlc.DeleteChannelModelParams{ChannelID: int64(channelID), ID: int64(modelID)})
	if err != nil {
		return false, mapError(err)
	}
	return affected > 0, nil
}

func (s *Store) ChannelModelExists(channelID int, modelName string) (bool, error) {
	_, err := s.queries.GetChannelModel(context.Background(), sqlc.GetChannelModelParams{ChannelID: int64(channelID), ModelName: modelName})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, mapError(err)
	}
	return true, nil
}

func (s *Store) ListCatalogModels(enabledOnly bool) (domain.ListResponse[domain.CatalogModelDTO], error) {
	rows, err := s.queries.ListCatalogModels(context.Background(), enabledOnly)
	if err != nil {
		return domain.ListResponse[domain.CatalogModelDTO]{}, mapError(err)
	}

	order := []string{}
	byName := map[string]*domain.CatalogModelDTO{}
	for _, row := range rows {
		entry := byName[row.ModelName]
		if entry == nil {
			entry = &domain.CatalogModelDTO{ModelName: row.ModelName, Status: 1}
			byName[row.ModelName] = entry
			order = append(order, row.ModelName)
		}
		entry.Channels = append(entry.Channels, domain.CatalogChannelDTO{ChannelID: int(row.ChannelID), ChannelName: row.ChannelName, UpstreamModel: row.UpstreamModel, Enabled: row.Enabled})
	}

	list := []domain.CatalogModelDTO{}
	for _, name := range order {
		list = append(list, *byName[name])
	}
	return domain.ListResponse[domain.CatalogModelDTO]{List: list, Total: len(list)}, nil
}

func (s *Store) ListPricing() (domain.ListResponse[domain.PricingDTO], error) {
	rows, err := s.queries.ListPricing(context.Background())
	if err != nil {
		return domain.ListResponse[domain.PricingDTO]{}, mapError(err)
	}
	list := []domain.PricingDTO{}
	for _, row := range rows {
		list = append(list, pricingDTO(row.ID, row.ChannelID, row.ChannelName, row.ModelName, textOrEmpty(row.UpstreamModel), row.InputPricePer1m, row.OutputPricePer1m, textValue(row.CachedInputPricePer1m), row.Currency))
	}
	return domain.ListResponse[domain.PricingDTO]{List: list, Total: len(list)}, nil
}

func (s *Store) UpsertPricingRecord(in domain.PricingRecord) (domain.PricingDTO, error) {
	ctx := context.Background()
	var cached any
	if in.CachedInputPricePer1M != "" {
		cached = in.CachedInputPricePer1M
	}
	if _, err := s.queries.UpsertPricing(ctx, sqlc.UpsertPricingParams{
		ChannelID:             int64(in.ChannelID),
		ModelName:             in.ModelName,
		InputPricePer1m:       in.InputPricePer1M,
		OutputPricePer1m:      in.OutputPricePer1M,
		CachedInputPricePer1m: cached,
		Currency:              in.Currency,
	}); err != nil {
		return domain.PricingDTO{}, mapError(err)
	}

	row, err := s.queries.GetPricing(ctx, sqlc.GetPricingParams{ChannelID: int64(in.ChannelID), ModelName: in.ModelName})
	if err != nil {
		return domain.PricingDTO{}, mapError(err)
	}
	return pricingDTO(row.ID, row.ChannelID, row.ChannelName, row.ModelName, textOrEmpty(row.UpstreamModel), row.InputPricePer1m, row.OutputPricePer1m, textValue(row.CachedInputPricePer1m), row.Currency), nil
}

func (s *Store) DeletePricing(in domain.DeletePricingInput) error {
	if err := s.queries.DeletePricing(context.Background(), sqlc.DeletePricingParams{
		ChannelID: int64(in.ChannelID),
		ModelName: in.ModelName,
	}); err != nil {
		return mapError(err)
	}
	return nil
}

func (s *Store) GetPricing(channelID int, modelName string) (domain.PricingDTO, error) {
	row, err := s.queries.GetPricing(context.Background(), sqlc.GetPricingParams{ChannelID: int64(channelID), ModelName: modelName})
	if err != nil {
		return domain.PricingDTO{}, mapError(err)
	}
	return pricingDTO(row.ID, row.ChannelID, row.ChannelName, row.ModelName, textOrEmpty(row.UpstreamModel), row.InputPricePer1m, row.OutputPricePer1m, textValue(row.CachedInputPricePer1m), row.Currency), nil
}

func (s *Store) RouteCandidates(modelName string, cooldownSeconds int) (domain.ListResponse[domain.RouteCandidate], error) {
	rows, err := s.queries.ListRouteCandidates(context.Background(), sqlc.ListRouteCandidatesParams{
		ModelName:       modelName,
		CooldownSeconds: int32(cooldownSeconds),
	})
	if err != nil {
		return domain.ListResponse[domain.RouteCandidate]{}, mapError(err)
	}
	list := []domain.RouteCandidate{}
	for _, row := range rows {
		list = append(list, domain.RouteCandidate{ChannelID: int(row.ChannelID), ChannelName: row.ChannelName, UpstreamModel: row.UpstreamModel, Priority: int(row.Priority), Weight: int(row.Weight), Balance: optionalString(textValue(row.Balance))})
	}
	return domain.ListResponse[domain.RouteCandidate]{List: list, Total: len(list)}, nil
}
