package storefake

import (
	"fmt"
	"sort"
	"time"

	"LLMGateway/server/internal/money"
	"LLMGateway/server/internal/store"
	domain "LLMGateway/server/internal/usage"
)

func (s *Store) InsertUsageLog(in domain.UsageLogInput) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.insertUsageLogLocked(in)
}

func (s *Store) insertUsageLogLocked(in domain.UsageLogInput) (int, error) {
	for _, log := range s.usageLogs {
		if log.RequestID == in.RequestID {
			return 0, fmt.Errorf("%w: duplicate request_id", store.ErrInvalid)
		}
	}

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

func (s *Store) ListUsageLogs(filter domain.UsageLogFilter) (domain.ListResponse[domain.UsageLogDTO], error) {
	if err := domain.ValidateTimeRange(filter.StartTime, filter.EndTime); err != nil {
		return domain.ListResponse[domain.UsageLogDTO]{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.filterUsageLogsLocked(filter)
	if err != nil {
		return domain.ListResponse[domain.UsageLogDTO]{}, err
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt != rows[j].CreatedAt {
			return rows[i].CreatedAt > rows[j].CreatedAt
		}
		return rows[i].ID > rows[j].ID
	})

	start, end := pageBounds(len(rows), filter.Page, filter.PageSize)
	list := []domain.UsageLogDTO{}
	for _, log := range rows[start:end] {
		list = append(list, usageLogDTO(&log))
	}
	return domain.ListResponse[domain.UsageLogDTO]{List: list, Total: len(rows)}, nil
}

func (s *Store) GetUsageLog(id int) (domain.UsageLogDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.usageLogs {
		if s.usageLogs[i].ID == id {
			return usageLogDTO(&s.usageLogs[i]), nil
		}
	}
	return domain.UsageLogDTO{}, store.ErrNotFound
}

func (s *Store) CountRequestsSince(filter domain.UsageCountFilter) (int, error) {
	if err := domain.ValidateSince(filter.Since); err != nil {
		return 0, err
	}
	sinceTime, _ := time.Parse(time.RFC3339, filter.Since)

	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for _, log := range s.usageLogs {
		if filter.UserID > 0 && (log.UserID == nil || *log.UserID != filter.UserID) {
			continue
		}
		if filter.APIKeyID != nil {
			if log.APIKeyID == nil || *log.APIKeyID != *filter.APIKeyID {
				continue
			}
		}
		if filter.Model != "" && log.Model != filter.Model {
			continue
		}
		if filter.ChannelID != nil {
			if log.ChannelID == nil || *log.ChannelID != *filter.ChannelID {
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

func (s *Store) CountTokensSince(filter domain.TokenCountFilter) (int64, error) {
	if err := domain.ValidateSince(filter.Since); err != nil {
		return 0, err
	}
	since, _ := time.Parse(time.RFC3339, filter.Since)
	s.mu.Lock()
	defer s.mu.Unlock()
	var total int64
	for _, log := range s.usageLogs {
		if (filter.UserID > 0 && (log.UserID == nil || *log.UserID != filter.UserID)) || (filter.APIKeyID != nil && (log.APIKeyID == nil || *log.APIKeyID != *filter.APIKeyID)) || (filter.Model != "" && log.Model != filter.Model) || (filter.ChannelID != nil && (log.ChannelID == nil || *log.ChannelID != *filter.ChannelID)) {
			continue
		}
		created, err := time.Parse(time.RFC3339, log.CreatedAt)
		if err == nil && !created.Before(since) {
			total += int64(log.TotalTokens)
		}
	}
	return total, nil
}

func (s *Store) StatsOverview(startTime, endTime string) (domain.StatsOverviewDTO, error) {
	start, end, err := parseRange(startTime, endTime)
	if err != nil {
		return domain.StatsOverviewDTO{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

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

	return domain.StatsOverviewDTO{RequestCount: int64(requests), SuccessCount: int64(success), ErrorCount: int64(errors), TotalTokens: int64(tokens), TotalCost: money.Format6(cost), ActiveUserCount: int64(len(users))}, nil
}

func (s *Store) StatsDaily(dateFrom, dateTo string, page, pageSize int) (domain.ListResponse[domain.StatsDailyDTO], error) {
	if err := domain.ValidateDateRange(dateFrom, dateTo); err != nil {
		return domain.ListResponse[domain.StatsDailyDTO]{}, err
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
	list := []domain.StatsDailyDTO{}
	for _, date := range dates[start:end] {
		entry := byDate[date]
		list = append(list, domain.StatsDailyDTO{StatDate: date, RequestCount: int64(entry.requests), SuccessCount: int64(entry.success), ErrorCount: int64(entry.errors), TotalTokens: int64(entry.tokens), TotalCost: money.Format6(entry.cost)})
	}
	return domain.ListResponse[domain.StatsDailyDTO]{List: list, Total: len(dates)}, nil
}

func (s *Store) StatsChannels(startTime, endTime string) (domain.ListResponse[domain.StatsChannelDTO], error) {
	start, end, err := parseRange(startTime, endTime)
	if err != nil {
		return domain.ListResponse[domain.StatsChannelDTO]{}, err
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

	list := []domain.StatsChannelDTO{}
	for _, channelID := range order {
		entry := byChannel[channelID]
		list = append(list, domain.StatsChannelDTO{ChannelID: channelID, ChannelName: entry.name, RequestCount: int64(entry.requests), SuccessCount: int64(entry.success), ErrorCount: int64(entry.errors), TotalTokens: int64(entry.tokens), TotalCost: money.Format6(entry.cost)})
	}
	return domain.ListResponse[domain.StatsChannelDTO]{List: list}, nil
}

func (s *Store) StatsTTFT(filter domain.TTFTStatsFilter) (domain.TTFTStatsDTO, error) {
	start, end, err := parseRange(filter.StartTime, filter.EndTime)
	if err != nil {
		return domain.TTFTStatsDTO{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var values []int
	for _, log := range s.usageLogs {
		if log.TTFTMs == nil || !withinRange(log.CreatedAt, start, end) ||
			(filter.UserID != nil && (log.UserID == nil || *log.UserID != *filter.UserID)) ||
			(filter.APIKeyID != nil && (log.APIKeyID == nil || *log.APIKeyID != *filter.APIKeyID)) ||
			(filter.ChannelID != nil && (log.ChannelID == nil || *log.ChannelID != *filter.ChannelID)) ||
			(filter.Model != "" && log.Model != filter.Model) {
			continue
		}
		values = append(values, *log.TTFTMs)
	}
	if len(values) == 0 {
		return domain.TTFTStatsDTO{}, nil
	}
	sort.Ints(values)
	var sum int64
	for _, value := range values {
		sum += int64(value)
	}
	return domain.TTFTStatsDTO{SampleCount: int64(len(values)), AverageMs: sum / int64(len(values)), P50Ms: percentile(values, 50), P95Ms: percentile(values, 95), P99Ms: percentile(values, 99)}, nil
}

func (s *Store) AggregateUsage(filter domain.UsageAggregateFilter) (domain.ListResponse[domain.UsageAggregateDTO], error) {
	start, end, err := parseRange(filter.StartTime, filter.EndTime)
	if err != nil {
		return domain.ListResponse[domain.UsageAggregateDTO]{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	byGroup := map[string]*domain.UsageAggregateDTO{}
	for _, log := range s.usageLogs {
		created, err := time.Parse(time.RFC3339, log.CreatedAt)
		if err != nil || (!start.IsZero() && created.Before(start)) || (!end.IsZero() && !created.Before(end)) || (filter.UserID != nil && (log.UserID == nil || *log.UserID != *filter.UserID)) || (filter.APIKeyID != nil && (log.APIKeyID == nil || *log.APIKeyID != *filter.APIKeyID)) || (filter.ChannelID != nil && (log.ChannelID == nil || *log.ChannelID != *filter.ChannelID)) || (filter.Model != "" && log.Model != filter.Model) || (filter.Status != "" && log.Status != filter.Status) {
			continue
		}
		key := ""
		item := domain.UsageAggregateDTO{}
		switch filter.GroupBy {
		case "user":
			item.UserID = log.UserID
			if log.UserID != nil {
				key = fmt.Sprintf("%d", *log.UserID)
			}
		case "api_key":
			item.APIKeyID = log.APIKeyID
			if log.APIKeyID != nil {
				key = fmt.Sprintf("%d", *log.APIKeyID)
			}
		case "model":
			item.Model = log.Model
			key = log.Model
		case "channel":
			item.ChannelID = log.ChannelID
			if log.ChannelID != nil {
				key = fmt.Sprintf("%d", *log.ChannelID)
			}
		default:
			return domain.ListResponse[domain.UsageAggregateDTO]{}, store.ErrInvalid
		}
		entry := byGroup[key]
		if entry == nil {
			entry = &item
			byGroup[key] = entry
		}
		entry.RequestCount++
		if log.Status == "success" {
			entry.SuccessCount++
		} else {
			entry.ErrorCount++
		}
		entry.TotalTokens += int64(log.TotalTokens)
		entry.DurationMs += int64(log.DurationMs)
		if cost, err := money.Parse6(log.TotalCost); err == nil {
			current, _ := money.Parse6(entry.TotalCost)
			entry.TotalCost = money.Format6(current.Add(cost))
		}
	}
	list := make([]domain.UsageAggregateDTO, 0, len(byGroup))
	for _, item := range byGroup {
		list = append(list, *item)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].RequestCount != list[j].RequestCount {
			return list[i].RequestCount > list[j].RequestCount
		}
		return aggregateKey(list[i], filter.GroupBy) < aggregateKey(list[j], filter.GroupBy)
	})
	startIndex, endIndex := pageBounds(len(list), filter.Page, filter.PageSize)
	return domain.ListResponse[domain.UsageAggregateDTO]{List: list[startIndex:endIndex], Total: len(list)}, nil
}

func aggregateKey(item domain.UsageAggregateDTO, groupBy string) string {
	switch groupBy {
	case "user":
		if item.UserID != nil {
			return fmt.Sprintf("%d", *item.UserID)
		}
	case "api_key":
		if item.APIKeyID != nil {
			return fmt.Sprintf("%d", *item.APIKeyID)
		}
	case "model":
		return item.Model
	case "channel":
		if item.ChannelID != nil {
			return fmt.Sprintf("%d", *item.ChannelID)
		}
	}
	return ""
}

func percentile(values []int, percentile int) int64 {
	index := (len(values)*percentile + 99) / 100
	return int64(values[index-1])
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
		if filter.APIKeyID != nil && (log.APIKeyID == nil || *log.APIKeyID != *filter.APIKeyID) {
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

func usageLogDTO(log *domain.UsageLog) domain.UsageLogDTO {
	return domain.UsageLogDTO{ID: log.ID, RequestID: log.RequestID, UserID: log.UserID, APIKeyID: log.APIKeyID, ChannelID: log.ChannelID, ChannelName: log.ChannelName, Model: log.Model, UpstreamModel: log.UpstreamModel, InputTokens: int64(log.InputTokens), OutputTokens: int64(log.OutputTokens), CachedInputTokens: int64(log.CachedInputTokens), TotalTokens: int64(log.TotalTokens), UnitPriceInputPer1M: log.UnitPriceInputPer1M, UnitPriceOutputPer1M: log.UnitPriceOutputPer1M, TotalCost: log.TotalCost, DurationMs: int64(log.DurationMs), TTFTMs: log.TTFTMs, Status: log.Status, ErrorCode: log.ErrorCode, ClientIP: log.ClientIP, CreatedAt: log.CreatedAt}
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
