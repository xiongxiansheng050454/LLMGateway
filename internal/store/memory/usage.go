package memory

import (
	"sort"
	"time"

	"LLMGateway/internal/domain"
	"LLMGateway/internal/money"
	"LLMGateway/internal/store"
)

func (s *Store) InsertUsageLog(in domain.UsageLogInput) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log := domain.UsageLog{
		ID:                   s.nextUsageLogID,
		RequestID:            in.RequestID,
		UserID:               in.UserID,
		APIKeyID:             in.APIKeyID,
		ChannelID:            in.ChannelID,
		ChannelName:          s.channelNameLocked(in.ChannelID),
		Model:                in.Model,
		UpstreamModel:        in.UpstreamModel,
		InputTokens:          in.InputTokens,
		OutputTokens:         in.OutputTokens,
		CachedInputTokens:    in.CachedInputTokens,
		TotalTokens:          in.TotalTokens,
		UnitPriceInputPer1M:  orZero8(in.UnitPriceInputPer1M),
		UnitPriceOutputPer1M: orZero8(in.UnitPriceOutputPer1M),
		TotalCost:            orZero6(in.TotalCost),
		DurationMs:           in.DurationMs,
		TTFTMs:               in.TTFTMs,
		Status:               in.Status,
		ErrorCode:            in.ErrorCode,
		ClientIP:             in.ClientIP,
		CreatedAt:            nowRFC3339(),
	}
	s.nextUsageLogID++
	s.usageLogs = append(s.usageLogs, log)
	return log.ID, nil
}

func (s *Store) ListUsageLogs(filter domain.UsageLogFilter) (domain.ListResponse, error) {
	if err := store.ValidateTimeRange(filter.StartTime, filter.EndTime); err != nil {
		return domain.ListResponse{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.filterUsageLogsLocked(filter)
	if err != nil {
		return domain.ListResponse{}, err
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt != rows[j].CreatedAt {
			return rows[i].CreatedAt > rows[j].CreatedAt
		}
		return rows[i].ID > rows[j].ID
	})

	start, end := pageBounds(len(rows), filter.Page, filter.PageSize)
	list := []any{}
	for _, log := range rows[start:end] {
		list = append(list, usageLogDTO(&log))
	}
	return domain.ListResponse{List: list, Total: len(rows)}, nil
}

func (s *Store) GetUsageLog(id int) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.usageLogs {
		if s.usageLogs[i].ID == id {
			return usageLogDTO(&s.usageLogs[i]), nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) CountRequestsSince(userID int, apiKeyID *int, since string) (int, error) {
	if err := store.ValidateSince(since); err != nil {
		return 0, err
	}
	sinceTime, _ := time.Parse(time.RFC3339, since)

	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for _, log := range s.usageLogs {
		if log.UserID == nil || *log.UserID != userID {
			continue
		}
		if apiKeyID != nil {
			if log.APIKeyID == nil || *log.APIKeyID != *apiKeyID {
				continue
			}
		}
		created, err := time.Parse(time.RFC3339, log.CreatedAt)
		if err != nil || created.Before(sinceTime) {
			continue
		}
		count++
	}
	return count, nil
}

func (s *Store) StatsOverview(startTime, endTime string) (map[string]any, error) {
	start, end, err := parseRange(startTime, endTime)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	overview := map[string]any{
		"request_count":     0,
		"success_count":     0,
		"error_count":       0,
		"total_tokens":      0,
		"total_cost":        "0.000000",
		"active_user_count": 0,
	}

	var requests, success, errors, tokens int
	cost := money.Amount(0)
	users := map[int]bool{}
	for _, log := range s.usageLogs {
		if !withinRange(log.CreatedAt, start, end) {
			continue
		}
		requests++
		if log.Status == "success" {
			success++
		} else {
			errors++
		}
		tokens += log.TotalTokens
		if parsed, err := money.Parse6(log.TotalCost); err == nil {
			cost = cost.Add(parsed)
		}
		if log.UserID != nil {
			users[*log.UserID] = true
		}
	}

	overview["request_count"] = requests
	overview["success_count"] = success
	overview["error_count"] = errors
	overview["total_tokens"] = tokens
	overview["total_cost"] = money.Format6(cost)
	overview["active_user_count"] = len(users)
	return overview, nil
}

func (s *Store) StatsDaily(dateFrom, dateTo string, page, pageSize int) (domain.ListResponse, error) {
	if err := store.ValidateDateRange(dateFrom, dateTo); err != nil {
		return domain.ListResponse{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	type bucket struct {
		requests, success, errors, tokens int
		cost                              money.Amount
	}
	byDate := map[string]*bucket{}
	for _, log := range s.usageLogs {
		date := utcDate(log.CreatedAt)
		if date == "" || date < dateFrom || date > dateTo {
			continue
		}
		entry := byDate[date]
		if entry == nil {
			entry = &bucket{}
			byDate[date] = entry
		}
		entry.requests++
		if log.Status == "success" {
			entry.success++
		} else {
			entry.errors++
		}
		entry.tokens += log.TotalTokens
		if parsed, err := money.Parse6(log.TotalCost); err == nil {
			entry.cost = entry.cost.Add(parsed)
		}
	}

	dates := make([]string, 0, len(byDate))
	for date := range byDate {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	start, end := pageBounds(len(dates), page, pageSize)
	list := []any{}
	for _, date := range dates[start:end] {
		entry := byDate[date]
		list = append(list, map[string]any{
			"stat_date":     date,
			"request_count": entry.requests,
			"success_count": entry.success,
			"error_count":   entry.errors,
			"total_tokens":  entry.tokens,
			"total_cost":    money.Format6(entry.cost),
		})
	}
	return domain.ListResponse{List: list, Total: len(dates)}, nil
}

func (s *Store) StatsChannels(startTime, endTime string) (domain.ListResponse, error) {
	start, end, err := parseRange(startTime, endTime)
	if err != nil {
		return domain.ListResponse{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	type bucket struct {
		name                              string
		requests, success, errors, tokens int
		cost                              money.Amount
	}
	byChannel := map[int]*bucket{}
	order := []int{}
	for _, log := range s.usageLogs {
		if !withinRange(log.CreatedAt, start, end) {
			continue
		}
		channelID := 0
		if log.ChannelID != nil {
			channelID = *log.ChannelID
		}
		entry := byChannel[channelID]
		if entry == nil {
			entry = &bucket{name: s.channelNameLocked(log.ChannelID)}
			byChannel[channelID] = entry
			order = append(order, channelID)
		}
		entry.requests++
		if log.Status == "success" {
			entry.success++
		} else {
			entry.errors++
		}
		entry.tokens += log.TotalTokens
		if parsed, err := money.Parse6(log.TotalCost); err == nil {
			entry.cost = entry.cost.Add(parsed)
		}
	}

	sort.Slice(order, func(i, j int) bool {
		if byChannel[order[i]].requests != byChannel[order[j]].requests {
			return byChannel[order[i]].requests > byChannel[order[j]].requests
		}
		return order[i] < order[j]
	})

	list := []any{}
	for _, channelID := range order {
		entry := byChannel[channelID]
		list = append(list, map[string]any{
			"channel_id":    channelID,
			"channel_name":  entry.name,
			"request_count": entry.requests,
			"success_count": entry.success,
			"error_count":   entry.errors,
			"total_tokens":  entry.tokens,
			"total_cost":    money.Format6(entry.cost),
		})
	}
	return domain.ListResponse{List: list}, nil
}

// orZero6/orZero8 mirror the PostgreSQL NOT NULL DEFAULT 0 columns so both
// stores return the same canonical strings for missing amounts.
func orZero6(value string) string {
	if value == "" {
		return "0.000000"
	}
	if parsed, err := money.Parse6(value); err == nil {
		return money.Format6(parsed)
	}
	return value
}

func orZero8(value string) string {
	if value == "" {
		return "0.00000000"
	}
	if parsed, err := money.Parse8(value); err == nil {
		return money.Format8(parsed)
	}
	return value
}

func (s *Store) filterUsageLogsLocked(filter domain.UsageLogFilter) ([]domain.UsageLog, error) {
	start, end, err := parseRange(filter.StartTime, filter.EndTime)
	if err != nil {
		return nil, err
	}
	rows := []domain.UsageLog{}
	for _, log := range s.usageLogs {
		if filter.UserID != nil && (log.UserID == nil || *log.UserID != *filter.UserID) {
			continue
		}
		if filter.ChannelID != nil && (log.ChannelID == nil || *log.ChannelID != *filter.ChannelID) {
			continue
		}
		if filter.Model != "" && log.Model != filter.Model {
			continue
		}
		if filter.Status != "" && log.Status != filter.Status {
			continue
		}
		if !withinRange(log.CreatedAt, start, end) {
			continue
		}
		rows = append(rows, log)
	}
	return rows, nil
}

func (s *Store) channelNameLocked(channelID *int) string {
	if channelID == nil {
		return ""
	}
	if channel, ok := s.channels[*channelID]; ok {
		return channel.Name
	}
	return ""
}

func usageLogDTO(log *domain.UsageLog) map[string]any {
	return map[string]any{
		"id":                       log.ID,
		"request_id":               log.RequestID,
		"user_id":                  log.UserID,
		"api_key_id":               log.APIKeyID,
		"channel_id":               log.ChannelID,
		"channel_name":             log.ChannelName,
		"model":                    log.Model,
		"upstream_model":           log.UpstreamModel,
		"input_tokens":             log.InputTokens,
		"output_tokens":            log.OutputTokens,
		"cached_input_tokens":      log.CachedInputTokens,
		"total_tokens":             log.TotalTokens,
		"unit_price_input_per_1m":  log.UnitPriceInputPer1M,
		"unit_price_output_per_1m": log.UnitPriceOutputPer1M,
		"total_cost":               log.TotalCost,
		"duration_ms":              log.DurationMs,
		"ttft_ms":                  log.TTFTMs,
		"status":                   log.Status,
		"error_code":               log.ErrorCode,
		"client_ip":                log.ClientIP,
		"created_at":               log.CreatedAt,
	}
}

func parseRange(startTime, endTime string) (time.Time, time.Time, error) {
	var start, end time.Time
	var err error
	if startTime != "" {
		start, err = time.Parse(time.RFC3339, startTime)
		if err != nil {
			return start, end, store.ErrInvalid
		}
	}
	if endTime != "" {
		end, err = time.Parse(time.RFC3339, endTime)
		if err != nil {
			return start, end, store.ErrInvalid
		}
	}
	return start, end, nil
}

func withinRange(createdAt string, start, end time.Time) bool {
	if start.IsZero() && end.IsZero() {
		return true
	}
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return false
	}
	if !start.IsZero() && created.Before(start) {
		return false
	}
	if !end.IsZero() && created.After(end) {
		return false
	}
	return true
}

func utcDate(createdAt string) string {
	parsed, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return ""
	}
	return parsed.UTC().Format("2006-01-02")
}
