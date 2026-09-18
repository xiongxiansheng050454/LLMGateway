package postgres

import (
	"context"
	"fmt"
	"time"

	"LLMGateway/server/internal/db/sqlc"
	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/money"
	"LLMGateway/server/internal/store"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Store) ListUsageLogs(filter domain.UsageLogFilter) (domain.ListResponse[domain.UsageLogDTO], error) {
	if err := domain.ValidateTimeRange(filter.StartTime, filter.EndTime); err != nil {
		return domain.ListResponse[domain.UsageLogDTO]{}, err
	}
	ctx := context.Background()
	limit, offset := limitOffset(filter.Page, filter.PageSize)
	params := sqlc.ListUsageLogsParams{
		UserID:     int8Value(filter.UserID),
		ApiKeyID:   int8Value(filter.APIKeyID),
		ChannelID:  int8Value(filter.ChannelID),
		Model:      textValueParam(filter.Model),
		Status:     textValueParam(filter.Status),
		StartTime:  timestampValue(filter.StartTime),
		EndTime:    timestampValue(filter.EndTime),
		PageOffset: offset,
		PageLimit:  limit,
	}

	rows, err := s.queries.ListUsageLogs(ctx, params)
	if err != nil {
		return domain.ListResponse[domain.UsageLogDTO]{}, mapError(err)
	}
	total, err := s.queries.CountUsageLogs(ctx, sqlc.CountUsageLogsParams{
		UserID:    params.UserID,
		ApiKeyID:  params.ApiKeyID,
		ChannelID: params.ChannelID,
		Model:     params.Model,
		Status:    params.Status,
		StartTime: params.StartTime,
		EndTime:   params.EndTime,
	})
	if err != nil {
		return domain.ListResponse[domain.UsageLogDTO]{}, mapError(err)
	}

	list := []domain.UsageLogDTO{}
	for _, row := range rows {
		list = append(list, usageLogDTO(row.ID, row.RequestID, row.UserID, row.ApiKeyID, row.ChannelID, row.ChannelName, row.Model, row.UpstreamModel, row.InputTokens, row.OutputTokens, row.CachedInputTokens, row.TotalTokens, row.UnitPriceInputPer1m, row.UnitPriceOutputPer1m, row.TotalCost, row.DurationMs, row.TtftMs, row.Status, row.ErrorCode, row.ClientIp, row.CreatedAt))
	}
	return domain.ListResponse[domain.UsageLogDTO]{List: list, Total: int(total)}, nil
}

func (s *Store) GetUsageLog(id int) (domain.UsageLogDTO, error) {
	row, err := s.queries.GetUsageLog(context.Background(), int64(id))
	if err != nil {
		return domain.UsageLogDTO{}, mapError(err)
	}
	return usageLogDTO(row.ID, row.RequestID, row.UserID, row.ApiKeyID, row.ChannelID, row.ChannelName, row.Model, row.UpstreamModel, row.InputTokens, row.OutputTokens, row.CachedInputTokens, row.TotalTokens, row.UnitPriceInputPer1m, row.UnitPriceOutputPer1m, row.TotalCost, row.DurationMs, row.TtftMs, row.Status, row.ErrorCode, row.ClientIp, row.CreatedAt), nil
}

func (s *Store) InsertUsageLog(in domain.UsageLogInput) (int, error) {
	id, err := insertUsageLog(context.Background(), s.queries, in)
	if err != nil {
		return 0, mapError(err)
	}
	return id, nil
}

func (s *Store) SettleChatCompletion(in domain.ChatSettlementInput) (int, error) {
	parsedCost, err := money.Parse6(in.Cost)
	if err != nil || parsedCost.Cmp(0) < 0 {
		return 0, fmt.Errorf("%w: invalid cost", store.ErrInvalid)
	}

	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, mapError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)
	if err := settleQuotaTx(ctx, tx, in, int64(in.UsageLog.TotalTokens), money.Format6(parsedCost), s.now()); err != nil {
		return 0, err
	}

	if _, err := queries.LockUserBalance(ctx, int64(in.UserID)); err != nil {
		return 0, mapError(err)
	}
	balanceRow, err := queries.GetUserBalanceText(ctx, int64(in.UserID))
	if err != nil {
		return 0, mapError(err)
	}
	current, err := money.Parse6(textValue(balanceRow.AvailableBalance))
	if err != nil {
		return 0, fmt.Errorf("%w: invalid balance", store.ErrInvalid)
	}
	if parsedCost.Cmp(0) > 0 && current.Cmp(parsedCost) < 0 {
		return 0, fmt.Errorf("%w: insufficient balance", store.ErrInvalid)
	}

	if parsedCost.Cmp(0) > 0 {
		next := money.Format6(current.Sub(parsedCost))
		if affected, err := queries.UpdateUserBalance(ctx, sqlc.UpdateUserBalanceParams{AvailableBalance: next, UserID: int64(in.UserID)}); err != nil {
			return 0, mapError(err)
		} else if affected == 0 {
			return 0, store.ErrNotFound
		}
		if _, err := queries.CreateBalanceTransaction(ctx, sqlc.CreateBalanceTransactionParams{UserID: int64(in.UserID), TxType: "consume", Amount: money.Format6(parsedCost), BalanceAfter: next, Description: in.Description}); err != nil {
			return 0, mapError(err)
		}
	}

	if in.DebitChannel && parsedCost.Cmp(0) > 0 {
		if in.ChannelID == nil {
			return 0, fmt.Errorf("%w: channel_id is required", store.ErrInvalid)
		}
		if _, err := queries.LockChannel(ctx, int64(*in.ChannelID)); err != nil {
			return 0, mapError(err)
		}
		row, err := queries.GetChannel(ctx, int64(*in.ChannelID))
		if err != nil {
			return 0, mapError(err)
		}
		base := money.Amount(0)
		if current := textValue(row.Balance); current != "" {
			parsed, err := money.Parse6(current)
			if err != nil {
				return 0, fmt.Errorf("%w: invalid channel balance", store.ErrInvalid)
			}
			base = parsed
		}
		if affected, err := queries.UpdateChannelBalance(ctx, sqlc.UpdateChannelBalanceParams{Balance: money.Format6(base.Sub(parsedCost)), ID: int64(*in.ChannelID)}); err != nil {
			return 0, mapError(err)
		} else if affected == 0 {
			return 0, store.ErrNotFound
		}
	}

	usageID, err := insertUsageLog(ctx, queries, in.UsageLog)
	if err != nil {
		return 0, mapError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, mapError(err)
	}
	return usageID, nil
}

func insertUsageLog(ctx context.Context, queries *sqlc.Queries, in domain.UsageLogInput) (int, error) {
	id, err := queries.InsertUsageLog(ctx, sqlc.InsertUsageLogParams{
		RequestID:            in.RequestID,
		UserID:               int8Value(in.UserID),
		ApiKeyID:             int8Value(in.APIKeyID),
		ChannelID:            int8Value(in.ChannelID),
		Model:                in.Model,
		UpstreamModel:        in.UpstreamModel,
		InputTokens:          int64(in.InputTokens),
		OutputTokens:         int64(in.OutputTokens),
		CachedInputTokens:    int64(in.CachedInputTokens),
		TotalTokens:          int64(in.TotalTokens),
		UnitPriceInputPer1m:  in.UnitPriceInputPer1M,
		UnitPriceOutputPer1m: in.UnitPriceOutputPer1M,
		TotalCost:            in.TotalCost,
		DurationMs:           int64(in.DurationMs),
		TtftMs:               int8Value(in.TTFTMs),
		Status:               in.Status,
		ErrorCode:            in.ErrorCode,
		ClientIp:             in.ClientIP,
	})
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (s *Store) CountRequestsSince(filter domain.UsageCountFilter) (int, error) {
	if err := domain.ValidateSince(filter.Since); err != nil {
		return 0, err
	}
	count, err := s.queries.CountRequestsSince(context.Background(), sqlc.CountRequestsSinceParams{
		UserID:    int8Value(&filter.UserID),
		Since:     timestampValue(filter.Since),
		ApiKeyID:  int8Value(filter.APIKeyID),
		Model:     textValueParam(filter.Model),
		ChannelID: int8Value(filter.ChannelID),
	})
	if err != nil {
		return 0, mapError(err)
	}
	return int(count), nil
}

func (s *Store) StatsOverview(startTime, endTime string) (domain.StatsOverviewDTO, error) {
	if err := domain.ValidateTimeRange(startTime, endTime); err != nil {
		return domain.StatsOverviewDTO{}, err
	}
	row, err := s.queries.StatsOverview(context.Background(), sqlc.StatsOverviewParams{
		CreatedAt:   timestampValue(startTime),
		CreatedAt_2: timestampValue(endTime),
	})
	if err != nil {
		return domain.StatsOverviewDTO{}, mapError(err)
	}
	return domain.StatsOverviewDTO{RequestCount: row.RequestCount, SuccessCount: row.SuccessCount, ErrorCount: row.ErrorCount, TotalTokens: row.TotalTokens, TotalCost: row.TotalCost, ActiveUserCount: row.ActiveUserCount}, nil
}

func (s *Store) StatsDaily(dateFrom, dateTo string, page, pageSize int) (domain.ListResponse[domain.StatsDailyDTO], error) {
	if err := domain.ValidateDateRange(dateFrom, dateTo); err != nil {
		return domain.ListResponse[domain.StatsDailyDTO]{}, err
	}
	ctx := context.Background()
	limit, offset := limitOffset(page, pageSize)

	rows, err := s.queries.StatsDaily(ctx, sqlc.StatsDailyParams{DateFrom: dateFrom, DateTo: dateTo, PageOffset: offset, PageLimit: limit})
	if err != nil {
		return domain.ListResponse[domain.StatsDailyDTO]{}, mapError(err)
	}
	total, err := s.queries.CountStatsDaily(ctx, sqlc.CountStatsDailyParams{DateFrom: dateFrom, DateTo: dateTo})
	if err != nil {
		return domain.ListResponse[domain.StatsDailyDTO]{}, mapError(err)
	}

	list := []domain.StatsDailyDTO{}
	for _, row := range rows {
		list = append(list, domain.StatsDailyDTO{StatDate: row.StatDate.Time.UTC().Format("2006-01-02"), RequestCount: row.RequestCount, SuccessCount: row.SuccessCount, ErrorCount: row.ErrorCount, TotalTokens: row.TotalTokens, TotalCost: row.TotalCost})
	}
	return domain.ListResponse[domain.StatsDailyDTO]{List: list, Total: int(total)}, nil
}

func (s *Store) StatsChannels(startTime, endTime string) (domain.ListResponse[domain.StatsChannelDTO], error) {
	if err := domain.ValidateTimeRange(startTime, endTime); err != nil {
		return domain.ListResponse[domain.StatsChannelDTO]{}, err
	}
	rows, err := s.queries.StatsChannels(context.Background(), sqlc.StatsChannelsParams{
		CreatedAt:   timestampValue(startTime),
		CreatedAt_2: timestampValue(endTime),
	})
	if err != nil {
		return domain.ListResponse[domain.StatsChannelDTO]{}, mapError(err)
	}

	list := []domain.StatsChannelDTO{}
	for _, row := range rows {
		channelID := 0
		if row.ChannelID.Valid {
			channelID = int(row.ChannelID.Int64)
		}
		list = append(list, domain.StatsChannelDTO{ChannelID: channelID, ChannelName: row.ChannelName, RequestCount: row.RequestCount, SuccessCount: row.SuccessCount, ErrorCount: row.ErrorCount, TotalTokens: row.TotalTokens, TotalCost: row.TotalCost})
	}
	return domain.ListResponse[domain.StatsChannelDTO]{List: list}, nil
}

func (s *Store) StatsTTFT(filter domain.TTFTStatsFilter) (domain.TTFTStatsDTO, error) {
	if err := domain.ValidateTimeRange(filter.StartTime, filter.EndTime); err != nil {
		return domain.TTFTStatsDTO{}, err
	}
	row, err := s.queries.StatsTTFT(context.Background(), sqlc.StatsTTFTParams{
		UserID:    int8Value(filter.UserID),
		ApiKeyID:  int8Value(filter.APIKeyID),
		ChannelID: int8Value(filter.ChannelID),
		Model:     textValueParam(filter.Model),
		StartTime: timestampValue(filter.StartTime),
		EndTime:   timestampValue(filter.EndTime),
	})
	if err != nil {
		return domain.TTFTStatsDTO{}, mapError(err)
	}
	return domain.TTFTStatsDTO{SampleCount: row.SampleCount, AverageMs: row.AverageMs, P50Ms: row.P50Ms, P95Ms: row.P95Ms, P99Ms: row.P99Ms}, nil
}

func (s *Store) AggregateUsage(filter domain.UsageAggregateFilter) (domain.ListResponse[domain.UsageAggregateDTO], error) {
	if err := domain.ValidateTimeRange(filter.StartTime, filter.EndTime); err != nil {
		return domain.ListResponse[domain.UsageAggregateDTO]{}, err
	}
	limit, offset := limitOffset(filter.Page, filter.PageSize)
	params := sqlc.AggregateUsageByModelParams{StartTime: timestampValue(filter.StartTime), EndTime: timestampValue(filter.EndTime), UserID: int8Value(filter.UserID), ApiKeyID: int8Value(filter.APIKeyID), ChannelID: int8Value(filter.ChannelID), Model: textValueParam(filter.Model), Status: textValueParam(filter.Status), PageOffset: offset, PageLimit: limit}
	ctx := context.Background()
	list := []domain.UsageAggregateDTO{}
	total := 0
	switch filter.GroupBy {
	case "user":
		rows, err := s.queries.AggregateUsageByUser(ctx, sqlc.AggregateUsageByUserParams(params))
		if err != nil {
			return domain.ListResponse[domain.UsageAggregateDTO]{}, mapError(err)
		}
		for _, row := range rows {
			list = append(list, domain.UsageAggregateDTO{UserID: optionalInt(row.UserID), RequestCount: row.RequestCount, SuccessCount: row.SuccessCount, ErrorCount: row.ErrorCount, TotalTokens: row.TotalTokens, TotalCost: row.TotalCost, DurationMs: row.DurationMs})
			total = int(row.AggregateTotal)
		}
	case "api_key":
		rows, err := s.queries.AggregateUsageByAPIKey(ctx, sqlc.AggregateUsageByAPIKeyParams(params))
		if err != nil {
			return domain.ListResponse[domain.UsageAggregateDTO]{}, mapError(err)
		}
		for _, row := range rows {
			list = append(list, domain.UsageAggregateDTO{APIKeyID: optionalInt(row.ApiKeyID), RequestCount: row.RequestCount, SuccessCount: row.SuccessCount, ErrorCount: row.ErrorCount, TotalTokens: row.TotalTokens, TotalCost: row.TotalCost, DurationMs: row.DurationMs})
			total = int(row.AggregateTotal)
		}
	case "model":
		rows, err := s.queries.AggregateUsageByModel(ctx, params)
		if err != nil {
			return domain.ListResponse[domain.UsageAggregateDTO]{}, mapError(err)
		}
		for _, row := range rows {
			list = append(list, domain.UsageAggregateDTO{Model: row.Model, RequestCount: row.RequestCount, SuccessCount: row.SuccessCount, ErrorCount: row.ErrorCount, TotalTokens: row.TotalTokens, TotalCost: row.TotalCost, DurationMs: row.DurationMs})
			total = int(row.AggregateTotal)
		}
	case "channel":
		rows, err := s.queries.AggregateUsageByChannel(ctx, sqlc.AggregateUsageByChannelParams(params))
		if err != nil {
			return domain.ListResponse[domain.UsageAggregateDTO]{}, mapError(err)
		}
		for _, row := range rows {
			list = append(list, domain.UsageAggregateDTO{ChannelID: optionalInt(row.ChannelID), RequestCount: row.RequestCount, SuccessCount: row.SuccessCount, ErrorCount: row.ErrorCount, TotalTokens: row.TotalTokens, TotalCost: row.TotalCost, DurationMs: row.DurationMs})
			total = int(row.AggregateTotal)
		}
	default:
		return domain.ListResponse[domain.UsageAggregateDTO]{}, store.ErrInvalid
	}
	return domain.ListResponse[domain.UsageAggregateDTO]{List: list, Total: total}, nil
}

func usageLogDTO(id int64, requestID string, userID, apiKeyID, channelID pgtype.Int8, channelName pgtype.Text, model, upstreamModel string, inputTokens, outputTokens, cachedInputTokens, totalTokens int64, unitPriceInput, unitPriceOutput, totalCost string, durationMs int64, ttftMs pgtype.Int8, status, errorCode, clientIP string, createdAt pgtype.Timestamptz) domain.UsageLogDTO {
	return domain.UsageLogDTO{ID: int(id), RequestID: requestID, UserID: optionalInt(userID), APIKeyID: optionalInt(apiKeyID), ChannelID: optionalInt(channelID), ChannelName: textOrEmpty(channelName), Model: model, UpstreamModel: upstreamModel, InputTokens: inputTokens, OutputTokens: outputTokens, CachedInputTokens: cachedInputTokens, TotalTokens: totalTokens, UnitPriceInputPer1M: unitPriceInput, UnitPriceOutputPer1M: unitPriceOutput, TotalCost: totalCost, DurationMs: durationMs, TTFTMs: optionalInt(ttftMs), Status: status, ErrorCode: errorCode, ClientIP: clientIP, CreatedAt: createdAt.Time.UTC().Format(time.RFC3339)}
}

func textValueParam(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}
