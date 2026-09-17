package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"LLMGateway/server/internal/db/sqlc"
	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/money"
	"LLMGateway/server/internal/store"
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

func (s *Store) CreateChannel(in domain.ChannelInput) (domain.ChannelDTO, error) {
	if strings.TrimSpace(in.APIKey) == "" {
		return domain.ChannelDTO{}, fmt.Errorf("%w: api_key is required", store.ErrInvalid)
	}
	if in.AuthType == "" {
		in.AuthType = "bearer"
	}
	if in.Weight == 0 {
		in.Weight = 100
	}

	balance, err := normalizeBalance(in.Balance)
	if err != nil {
		return domain.ChannelDTO{}, err
	}
	ciphertext, err := s.encryptSecret(in.APIKey)
	if err != nil {
		return domain.ChannelDTO{}, err
	}

	id, err := s.queries.CreateChannel(context.Background(), sqlc.CreateChannelParams{
		Name:             in.Name,
		BaseUrl:          in.BaseURL,
		ApiKeyCiphertext: ciphertext,
		AuthType:         in.AuthType,
		Status:           int32(in.Status),
		Weight:           int32(in.Weight),
		Priority:         int32(in.Priority),
		Balance:          balance,
	})
	if err != nil {
		return domain.ChannelDTO{}, mapError(err)
	}
	return s.getChannelDTO(id)
}

func (s *Store) UpdateChannel(id int, in domain.ChannelInput) (domain.ChannelDTO, error) {
	balance, err := normalizeBalance(in.Balance)
	if err != nil {
		return domain.ChannelDTO{}, err
	}

	var ciphertext any = ""
	if strings.TrimSpace(in.APIKey) != "" {
		encrypted, err := s.encryptSecret(in.APIKey)
		if err != nil {
			return domain.ChannelDTO{}, err
		}
		ciphertext = encrypted
	}

	affected, err := s.queries.UpdateChannel(context.Background(), sqlc.UpdateChannelParams{
		Name:             in.Name,
		BaseUrl:          in.BaseURL,
		AuthType:         in.AuthType,
		Status:           int32(in.Status),
		Weight:           int32(in.Weight),
		Priority:         int32(in.Priority),
		Balance:          balance,
		ApiKeyCiphertext: ciphertext,
		ID:               int64(id),
	})
	if err != nil {
		return domain.ChannelDTO{}, mapError(err)
	}
	if affected == 0 {
		return domain.ChannelDTO{}, store.ErrNotFound
	}
	return s.getChannelDTO(int64(id))
}

func (s *Store) UpdateChannelStatus(id int, status int) (domain.ChannelDTO, error) {
	affected, err := s.queries.UpdateChannelStatus(context.Background(), sqlc.UpdateChannelStatusParams{
		Status: int32(status),
		ID:     int64(id),
	})
	if err != nil {
		return domain.ChannelDTO{}, mapError(err)
	}
	if affected == 0 {
		return domain.ChannelDTO{}, store.ErrNotFound
	}
	return s.getChannelDTO(int64(id))
}

func (s *Store) UpdateChannelBalance(id int, balance string, delta string) (domain.ChannelDTO, error) {
	if balance == "" && delta == "" {
		return domain.ChannelDTO{}, fmt.Errorf("%w: balance or delta is required", store.ErrInvalid)
	}

	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ChannelDTO{}, mapError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)

	// Lock the row so concurrent read-modify-write balance updates cannot be
	// lost under READ COMMITTED.
	if _, err := queries.LockChannel(ctx, int64(id)); err != nil {
		return domain.ChannelDTO{}, mapError(err)
	}
	row, err := queries.GetChannel(ctx, int64(id))
	if err != nil {
		return domain.ChannelDTO{}, mapError(err)
	}

	base := money.Amount(0)
	if current := textValue(row.Balance); current != "" {
		parsed, err := money.Parse6(current)
		if err != nil {
			return domain.ChannelDTO{}, fmt.Errorf("%w: invalid balance", store.ErrInvalid)
		}
		base = parsed
	}
	if balance != "" {
		parsed, err := money.Parse6(balance)
		if err != nil {
			return domain.ChannelDTO{}, fmt.Errorf("%w: invalid balance", store.ErrInvalid)
		}
		base = parsed
	}
	if delta != "" {
		parsed, err := money.Parse6(delta)
		if err != nil {
			return domain.ChannelDTO{}, fmt.Errorf("%w: invalid delta", store.ErrInvalid)
		}
		base = base.Add(parsed)
	}

	affected, err := queries.UpdateChannelBalance(ctx, sqlc.UpdateChannelBalanceParams{
		Balance: money.Format6(base),
		ID:      int64(id),
	})
	if err != nil {
		return domain.ChannelDTO{}, mapError(err)
	}
	if affected == 0 {
		return domain.ChannelDTO{}, store.ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.ChannelDTO{}, mapError(err)
	}
	return s.getChannelDTO(int64(id))
}

func (s *Store) DeleteChannel(id int) error {
	affected, err := s.queries.DeleteChannel(context.Background(), int64(id))
	if err != nil {
		return mapError(err)
	}
	if affected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *Store) GetChannelSecret(id int) (*domain.Channel, error) {
	row, err := s.queries.GetChannelSecret(context.Background(), int64(id))
	if err != nil {
		return nil, mapError(err)
	}

	plaintext := ""
	if row.ApiKeyCiphertext != "" {
		if s.cipher == nil {
			return nil, errors.New("channel encryption key is not configured")
		}
		decrypted, err := s.cipher.Decrypt(row.ApiKeyCiphertext)
		if err != nil {
			return nil, fmt.Errorf("decrypt channel api_key: %w", err)
		}
		plaintext = decrypted
	}

	return &domain.Channel{
		ID:       int(row.ID),
		Name:     row.Name,
		BaseURL:  row.BaseUrl,
		APIKey:   plaintext,
		AuthType: row.AuthType,
		Status:   int(row.Status),
		Weight:   int(row.Weight),
		Priority: int(row.Priority),
		Balance:  optionalString(textValue(row.Balance)),
	}, nil
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

func (s *Store) CreateChannelModel(channelID int, in domain.ChannelModel) (domain.ChannelModel, error) {
	ctx := context.Background()
	if _, err := s.queries.GetChannel(ctx, int64(channelID)); err != nil {
		return domain.ChannelModel{}, mapError(err)
	}
	row, err := s.queries.CreateChannelModel(ctx, sqlc.CreateChannelModelParams{
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

func (s *Store) UpdateChannelModel(channelID, modelID int, upstreamModel string, enabled bool) (domain.ChannelModel, error) {
	row, err := s.queries.UpdateChannelModel(context.Background(), sqlc.UpdateChannelModelParams{
		UpstreamModel: upstreamModel,
		Enabled:       enabled,
		ChannelID:     int64(channelID),
		ID:            int64(modelID),
	})
	if err != nil {
		return domain.ChannelModel{}, mapError(err)
	}
	return domain.ChannelModel{ID: int(row.ID), ModelName: row.ModelName, UpstreamModel: row.UpstreamModel, Enabled: row.Enabled}, nil
}

func (s *Store) DeleteChannelModel(channelID, modelID int) error {
	affected, err := s.queries.DeleteChannelModel(context.Background(), sqlc.DeleteChannelModelParams{
		ChannelID: int64(channelID),
		ID:        int64(modelID),
	})
	if err != nil {
		return mapError(err)
	}
	if affected == 0 {
		return store.ErrNotFound
	}
	return nil
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

func (s *Store) UpsertPricing(in domain.PricingInput) (domain.PricingDTO, error) {
	if in.ChannelID <= 0 || strings.TrimSpace(in.ModelName) == "" {
		return domain.PricingDTO{}, fmt.Errorf("%w: channel_id and model_name are required", store.ErrInvalid)
	}
	ctx := context.Background()
	if _, err := s.queries.GetChannel(ctx, int64(in.ChannelID)); err != nil {
		return domain.PricingDTO{}, mapError(err)
	}
	if _, err := s.queries.GetChannelModel(ctx, sqlc.GetChannelModelParams{ChannelID: int64(in.ChannelID), ModelName: in.ModelName}); err != nil {
		if errors.Is(mapError(err), store.ErrNotFound) {
			return domain.PricingDTO{}, fmt.Errorf("%w: model mapping not found", store.ErrInvalid)
		}
		return domain.PricingDTO{}, mapError(err)
	}

	inputPrice, err := normalizePrice(in.InputPricePer1M, "input_price_per_1m")
	if err != nil {
		return domain.PricingDTO{}, err
	}
	outputPrice, err := normalizePrice(in.OutputPricePer1M, "output_price_per_1m")
	if err != nil {
		return domain.PricingDTO{}, err
	}
	cachedPrice, err := normalizeOptionalPrice(in.CachedInputPricePer1M, "cached_input_price_per_1m")
	if err != nil {
		return domain.PricingDTO{}, err
	}
	currency := in.Currency
	if currency == "" {
		currency = "USD"
	}

	if _, err := s.queries.UpsertPricing(ctx, sqlc.UpsertPricingParams{
		ChannelID:             int64(in.ChannelID),
		ModelName:             in.ModelName,
		InputPricePer1m:       inputPrice,
		OutputPricePer1m:      outputPrice,
		CachedInputPricePer1m: cachedPrice,
		Currency:              currency,
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

func (s *Store) RouteCandidates(modelName string) (domain.ListResponse[domain.RouteCandidate], error) {
	rows, err := s.queries.ListRouteCandidates(context.Background(), sqlc.ListRouteCandidatesParams{
		ModelName:       modelName,
		CooldownSeconds: int32(s.breaker.Cooldown.Seconds()),
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

func (s *Store) TestChannel(channelID int) (domain.ChannelTestResultDTO, error) {
	ctx := context.Background()
	if _, err := s.queries.GetChannel(ctx, int64(channelID)); err != nil {
		return domain.ChannelTestResultDTO{}, mapError(err)
	}
	rows, err := s.queries.ListChannelModels(ctx, int64(channelID))
	if err != nil {
		return domain.ChannelTestResultDTO{}, mapError(err)
	}
	items := []domain.ChannelTestItemDTO{}
	for _, row := range rows {
		items = append(items, domain.ChannelTestItemDTO{ModelAlias: row.ModelName, UpstreamModel: row.UpstreamModel, Error: "not tested in MVP"})
	}
	return domain.ChannelTestResultDTO{List: items}, nil
}

func (s *Store) getChannelDTO(id int64) (domain.ChannelDTO, error) {
	row, err := s.queries.GetChannel(context.Background(), id)
	if err != nil {
		return domain.ChannelDTO{}, mapError(err)
	}
	return channelDTO(row.ID, row.Name, row.BaseUrl, row.AuthType, row.Status, row.Weight, row.Priority, textValue(row.Balance), row.ModelCount), nil
}

func (s *Store) encryptSecret(plaintext string) (string, error) {
	if s.cipher == nil {
		return "", errors.New("channel encryption key is not configured")
	}
	return s.cipher.Encrypt(plaintext)
}

func normalizeBalance(balance *string) (any, error) {
	if balance == nil || *balance == "" {
		return nil, nil
	}
	amount, err := money.Parse6(*balance)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid balance", store.ErrInvalid)
	}
	return money.Format6(amount), nil
}

func normalizePrice(value string, field string) (any, error) {
	amount, err := money.Parse8(value)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid %s", store.ErrInvalid, field)
	}
	return money.Format8(amount), nil
}

func normalizeOptionalPrice(value string, field string) (any, error) {
	if value == "" {
		return nil, nil
	}
	return normalizePrice(value, field)
}
