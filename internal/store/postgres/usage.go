package postgres

import (
	"context"
	"time"

	"LLMGateway/internal/db/sqlc"
	"LLMGateway/internal/domain"
	"LLMGateway/internal/store"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Store) ListUsageLogs(filter domain.UsageLogFilter) (domain.ListResponse, error) {
	if err := store.ValidateTimeRange(filter.StartTime, filter.EndTime); err != nil {
		return domain.ListResponse{}, err
	}
	ctx := context.Background()
	limit, offset := limitOffset(filter.Page, filter.PageSize)
	params := sqlc.ListUsageLogsParams{
		UserID:     int8Value(filter.UserID),
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
		return domain.ListResponse{}, mapError(err)
	}
	total, err := s.queries.CountUsageLogs(ctx, sqlc.CountUsageLogsParams{
		UserID:    params.UserID,
		ChannelID: params.ChannelID,
		Model:     params.Model,
		Status:    params.Status,
		StartTime: params.StartTime,
		EndTime:   params.EndTime,
	})
	if err != nil {
		return domain.ListResponse{}, mapError(err)
	}

	list := []any{}
	for _, row := range rows {
		list = append(list, usageLogDTO(row.ID, row.RequestID, row.UserID, row.ApiKeyID, row.ChannelID, row.ChannelName, row.Model, row.UpstreamModel, row.InputTokens, row.OutputTokens, row.CachedInputTokens, row.TotalTokens, row.UnitPriceInputPer1m, row.UnitPriceOutputPer1m, row.TotalCost, row.DurationMs, row.TtftMs, row.Status, row.ErrorCode, row.ClientIp, row.CreatedAt))
	}
	return domain.ListResponse{List: list, Total: int(total)}, nil
}

func (s *Store) GetUsageLog(id int) (map[string]any, error) {
	row, err := s.queries.GetUsageLog(context.Background(), int64(id))
	if err != nil {
		return nil, mapError(err)
	}
	return usageLogDTO(row.ID, row.RequestID, row.UserID, row.ApiKeyID, row.ChannelID, row.ChannelName, row.Model, row.UpstreamModel, row.InputTokens, row.OutputTokens, row.CachedInputTokens, row.TotalTokens, row.UnitPriceInputPer1m, row.UnitPriceOutputPer1m, row.TotalCost, row.DurationMs, row.TtftMs, row.Status, row.ErrorCode, row.ClientIp, row.CreatedAt), nil
}

func (s *Store) InsertUsageLog(in domain.UsageLogInput) (int, error) {
	id, err := s.queries.InsertUsageLog(context.Background(), sqlc.InsertUsageLogParams{
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
		return 0, mapError(err)
	}
	return int(id), nil
}

func (s *Store) CountRequestsSince(userID int, apiKeyID *int, since string) (int, error) {
	if err := store.ValidateSince(since); err != nil {
		return 0, err
	}
	count, err := s.queries.CountRequestsSince(context.Background(), sqlc.CountRequestsSinceParams{
		UserID:   int8Value(&userID),
		Since:    timestampValue(since),
		ApiKeyID: int8Value(apiKeyID),
	})
	if err != nil {
		return 0, mapError(err)
	}
	return int(count), nil
}

func (s *Store) StatsOverview(startTime, endTime string) (map[string]any, error) {
	if err := store.ValidateTimeRange(startTime, endTime); err != nil {
		return nil, err
	}
	row, err := s.queries.StatsOverview(context.Background(), sqlc.StatsOverviewParams{
		CreatedAt:   timestampValue(startTime),
		CreatedAt_2: timestampValue(endTime),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return map[string]any{
		"request_count":     row.RequestCount,
		"success_count":     row.SuccessCount,
		"error_count":       row.ErrorCount,
		"total_tokens":      row.TotalTokens,
		"total_cost":        row.TotalCost,
		"active_user_count": row.ActiveUserCount,
	}, nil
}

func (s *Store) StatsDaily(dateFrom, dateTo string, page, pageSize int) (domain.ListResponse, error) {
	if err := store.ValidateDateRange(dateFrom, dateTo); err != nil {
		return domain.ListResponse{}, err
	}
	ctx := context.Background()
	limit, offset := limitOffset(page, pageSize)

	rows, err := s.queries.StatsDaily(ctx, sqlc.StatsDailyParams{DateFrom: dateFrom, DateTo: dateTo, PageOffset: offset, PageLimit: limit})
	if err != nil {
		return domain.ListResponse{}, mapError(err)
	}
	total, err := s.queries.CountStatsDaily(ctx, sqlc.CountStatsDailyParams{DateFrom: dateFrom, DateTo: dateTo})
	if err != nil {
		return domain.ListResponse{}, mapError(err)
	}

	list := []any{}
	for _, row := range rows {
		list = append(list, map[string]any{
			"stat_date":     row.StatDate.Time.UTC().Format("2006-01-02"),
			"request_count": row.RequestCount,
			"success_count": row.SuccessCount,
			"error_count":   row.ErrorCount,
			"total_tokens":  row.TotalTokens,
			"total_cost":    row.TotalCost,
		})
	}
	return domain.ListResponse{List: list, Total: int(total)}, nil
}

func (s *Store) StatsChannels(startTime, endTime string) (domain.ListResponse, error) {
	if err := store.ValidateTimeRange(startTime, endTime); err != nil {
		return domain.ListResponse{}, err
	}
	rows, err := s.queries.StatsChannels(context.Background(), sqlc.StatsChannelsParams{
		CreatedAt:   timestampValue(startTime),
		CreatedAt_2: timestampValue(endTime),
	})
	if err != nil {
		return domain.ListResponse{}, mapError(err)
	}

	list := []any{}
	for _, row := range rows {
		channelID := 0
		if row.ChannelID.Valid {
			channelID = int(row.ChannelID.Int64)
		}
		list = append(list, map[string]any{
			"channel_id":    channelID,
			"channel_name":  row.ChannelName,
			"request_count": row.RequestCount,
			"success_count": row.SuccessCount,
			"error_count":   row.ErrorCount,
			"total_tokens":  row.TotalTokens,
			"total_cost":    row.TotalCost,
		})
	}
	return domain.ListResponse{List: list}, nil
}

func usageLogDTO(id int64, requestID string, userID, apiKeyID, channelID pgtype.Int8, channelName pgtype.Text, model, upstreamModel string, inputTokens, outputTokens, cachedInputTokens, totalTokens int64, unitPriceInput, unitPriceOutput, totalCost string, durationMs int64, ttftMs pgtype.Int8, status, errorCode, clientIP string, createdAt pgtype.Timestamptz) map[string]any {
	return map[string]any{
		"id":                       int(id),
		"request_id":               requestID,
		"user_id":                  optionalInt(userID),
		"api_key_id":               optionalInt(apiKeyID),
		"channel_id":               optionalInt(channelID),
		"channel_name":             textOrEmpty(channelName),
		"model":                    model,
		"upstream_model":           upstreamModel,
		"input_tokens":             inputTokens,
		"output_tokens":            outputTokens,
		"cached_input_tokens":      cachedInputTokens,
		"total_tokens":             totalTokens,
		"unit_price_input_per_1m":  unitPriceInput,
		"unit_price_output_per_1m": unitPriceOutput,
		"total_cost":               totalCost,
		"duration_ms":              durationMs,
		"ttft_ms":                  optionalInt(ttftMs),
		"status":                   status,
		"error_code":               errorCode,
		"client_ip":                clientIP,
		"created_at":               createdAt.Time.UTC().Format(time.RFC3339),
	}
}

func textValueParam(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}
