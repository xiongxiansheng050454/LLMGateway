package usage

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"LLMGateway/server/internal/httpcommon"
)

const (
	defaultStartTime = "1970-01-01T00:00:00Z"
	defaultEndTime   = "9999-12-31T23:59:59Z"
	defaultDateFrom  = "1970-01-01"
	defaultDateTo    = "9999-12-31"
)

func (a *Server) Data(r *http.Request) httpcommon.AdminResult {
	parts := httpcommon.SplitPath(strings.TrimSuffix(r.URL.Path, "/"))
	if len(parts) < 2 || parts[0] != "admin" {
		return httpcommon.Unhandled()
	}

	if parts[1] == "usage-logs" {
		if len(parts) == 2 && r.Method == http.MethodGet {
			return a.listUsageLogs(r)
		}
		if len(parts) == 3 && r.Method == http.MethodGet {
			id, err := strconv.Atoi(parts[2])
			if err != nil {
				return httpcommon.HTTPError(http.StatusBadRequest, "invalid log id")
			}
			return httpcommon.Result(a.store.GetUsageLog(id))
		}
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}

	if parts[1] == "stats" && len(parts) == 3 && r.Method == http.MethodGet {
		switch parts[2] {
		case "overview":
			return a.statsOverview(r)
		case "daily":
			return a.statsDaily(r)
		case "channels":
			return a.statsChannels(r)
		case "ttft":
			return a.statsTTFT(r)
		case "usage":
			return a.aggregateUsage(r)
		}
	}
	return httpcommon.Unhandled()
}

func (a *Server) listUsageLogs(r *http.Request) httpcommon.AdminResult {
	query := r.URL.Query()
	page, pageSize := httpcommon.ParsePagination(r)

	filter := UsageLogFilter{
		Model:     query.Get("model"),
		Status:    query.Get("status"),
		StartTime: query.Get("start_time"),
		EndTime:   query.Get("end_time"),
		Page:      page,
		PageSize:  pageSize,
	}
	if value := query.Get("user_id"); value != "" {
		id, err := strconv.Atoi(value)
		if err != nil {
			return httpcommon.HTTPError(http.StatusBadRequest, "invalid user_id")
		}
		filter.UserID = &id
	}
	if value := query.Get("channel_id"); value != "" {
		id, err := strconv.Atoi(value)
		if err != nil {
			return httpcommon.HTTPError(http.StatusBadRequest, "invalid channel_id")
		}
		filter.ChannelID = &id
	}
	if value := query.Get("api_key_id"); value != "" {
		id, err := strconv.Atoi(value)
		if err != nil || id <= 0 {
			return httpcommon.HTTPError(http.StatusBadRequest, "invalid api_key_id")
		}
		filter.APIKeyID = &id
	}
	return httpcommon.Result(a.store.ListUsageLogs(filter))
}

func (a *Server) statsOverview(r *http.Request) httpcommon.AdminResult {
	query := r.URL.Query()
	return httpcommon.Result(a.store.StatsOverview(orDefault(query.Get("start_time"), defaultStartTime), orDefault(query.Get("end_time"), defaultEndTime)))
}

func (a *Server) statsDaily(r *http.Request) httpcommon.AdminResult {
	query := r.URL.Query()
	page, pageSize := httpcommon.ParsePagination(r)
	return httpcommon.Result(a.store.StatsDaily(orDefault(query.Get("date_from"), defaultDateFrom), orDefault(query.Get("date_to"), defaultDateTo), page, pageSize))
}

func (a *Server) statsChannels(r *http.Request) httpcommon.AdminResult {
	query := r.URL.Query()
	return httpcommon.Result(a.store.StatsChannels(orDefault(query.Get("start_time"), defaultStartTime), orDefault(query.Get("end_time"), defaultEndTime)))
}

func (a *Server) statsTTFT(r *http.Request) httpcommon.AdminResult {
	query := r.URL.Query()
	filter := TTFTStatsFilter{Model: query.Get("model"), StartTime: orDefault(query.Get("start_time"), defaultStartTime), EndTime: orDefault(query.Get("end_time"), defaultEndTime)}
	for name, target := range map[string]**int{"user_id": &filter.UserID, "api_key_id": &filter.APIKeyID, "channel_id": &filter.ChannelID} {
		if value := query.Get(name); value != "" {
			id, err := strconv.Atoi(value)
			if err != nil {
				return httpcommon.HTTPError(http.StatusBadRequest, "invalid "+name)
			}
			*target = &id
		}
	}
	return httpcommon.Result(a.store.StatsTTFT(filter))
}

func (a *Server) aggregateUsage(r *http.Request) httpcommon.AdminResult {
	query := r.URL.Query()
	groupBy := query.Get("group_by")
	if groupBy != "user" && groupBy != "api_key" && groupBy != "model" && groupBy != "channel" {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid group_by")
	}
	if (query.Get("start_time") != "" || query.Get("end_time") != "") && (query.Get("date_from") != "" || query.Get("date_to") != "") {
		return httpcommon.HTTPError(http.StatusBadRequest, "time and date ranges cannot be combined")
	}
	startTime, endTime := query.Get("start_time"), query.Get("end_time")
	if query.Get("date_from") != "" || query.Get("date_to") != "" {
		if err := ValidateDateRange(query.Get("date_from"), query.Get("date_to")); err != nil {
			return httpcommon.HTTPError(http.StatusBadRequest, "invalid date range")
		}
		startTime = orDefault(query.Get("date_from"), defaultDateFrom) + "T00:00:00Z"
		if query.Get("date_to") == "" {
			endTime = defaultEndTime
		} else {
			end, _ := time.Parse("2006-01-02", query.Get("date_to"))
			endTime = end.AddDate(0, 0, 1).Format(time.RFC3339)
		}
	}
	filter := UsageAggregateFilter{GroupBy: groupBy, Model: query.Get("model"), Status: query.Get("status"), StartTime: orDefault(startTime, defaultStartTime), EndTime: orDefault(endTime, defaultEndTime)}
	filter.Page, filter.PageSize = httpcommon.ParsePagination(r)
	for name, target := range map[string]**int{"user_id": &filter.UserID, "api_key_id": &filter.APIKeyID, "channel_id": &filter.ChannelID} {
		if value := query.Get(name); value != "" {
			id, err := strconv.Atoi(value)
			if err != nil || id <= 0 {
				return httpcommon.HTTPError(http.StatusBadRequest, "invalid "+name)
			}
			*target = &id
		}
	}
	return httpcommon.Result(a.store.AggregateUsage(filter))
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
