package memory

import (
	"fmt"
	"sort"
	"time"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/money"
	"LLMGateway/server/internal/store"
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

func (s *Store) SettleChatCompletion(in domain.ChatSettlementInput) (int, error) {
	parsedCost, err := money.Parse6(in.Cost)
	if err != nil || parsedCost.Cmp(0) < 0 {
		return 0, fmt.Errorf("%w: invalid cost", store.ErrInvalid)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[in.UserID]
	if !ok {
		return 0, store.ErrNotFound
	}
	if in.DebitChannel {
		if in.ChannelID == nil {
			return 0, fmt.Errorf("%w: channel_id is required", store.ErrInvalid)
		}
		if _, ok := s.channels[*in.ChannelID]; !ok {
			return 0, store.ErrNotFound
		}
	}
	for _, log := range s.usageLogs {
		if log.RequestID == in.UsageLog.RequestID {
			return 0, fmt.Errorf("%w: duplicate request_id", store.ErrInvalid)
		}
	}

	currentUserBalance, err := money.Parse6(user.AvailableBalance)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid balance", store.ErrInvalid)
	}
	if parsedCost.Cmp(0) > 0 && currentUserBalance.Cmp(parsedCost) < 0 {
		return 0, fmt.Errorf("%w: insufficient balance", store.ErrInvalid)
	}

	var channel *domain.Channel
	var nextChannelBalance *string
	if in.DebitChannel && parsedCost.Cmp(0) > 0 {
		channel = s.channels[*in.ChannelID]
		base := money.Amount(0)
		if channel.Balance != nil {
			parsed, err := money.Parse6(*channel.Balance)
			if err != nil {
				return 0, fmt.Errorf("%w: invalid channel balance", store.ErrInvalid)
			}
			base = parsed
		}
		formatted := money.Format6(base.Sub(parsedCost))
		nextChannelBalance = &formatted
	}

	if parsedCost.Cmp(0) > 0 {
		next := money.Format6(currentUserBalance.Sub(parsedCost))
		user.AvailableBalance = next
		tx := domain.BalanceTransaction{ID: s.nextTxID, TxType: "consume", Amount: money.Format6(parsedCost), BalanceAfter: next, Description: in.Description, CreatedAt: nowRFC3339()}
		s.nextTxID++
		s.transactions[in.UserID] = append(s.transactions[in.UserID], tx)
	}
	if channel != nil {
		channel.Balance = nextChannelBalance
	}
	return s.insertUsageLogLocked(in.UsageLog)
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

func (s *Store) CountRequestsSince(userID int, apiKeyID *int, since string) (int, error) {
	if err := domain.ValidateSince(since); err != nil {
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
