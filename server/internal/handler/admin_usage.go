package handler

import (
	"net/http"
	"strconv"
	"strings"

	"LLMGateway/server/internal/domain"
)

const (
	defaultStartTime = "1970-01-01T00:00:00Z"
	defaultEndTime   = "9999-12-31T23:59:59Z"
	defaultDateFrom  = "1970-01-01"
	defaultDateTo    = "9999-12-31"
)

func (a *Server) usageData(r *http.Request) (any, bool, int, string) {
	parts := splitPath(strings.TrimSuffix(r.URL.Path, "/"))
	if len(parts) < 2 || parts[0] != "admin" {
		return nil, false, 0, ""
	}

	if parts[1] == "usage-logs" {
		if len(parts) == 2 && r.Method == http.MethodGet {
			return a.listUsageLogs(r)
		}
		if len(parts) == 3 && r.Method == http.MethodGet {
			id, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, true, http.StatusBadRequest, "invalid log id"
			}
			return a.result(a.store.GetUsageLog(id))
		}
		return nil, true, http.StatusMethodNotAllowed, "method not allowed"
	}

	if parts[1] == "stats" && len(parts) == 3 && r.Method == http.MethodGet {
		switch parts[2] {
		case "overview":
			return a.statsOverview(r)
		case "daily":
			return a.statsDaily(r)
		case "channels":
			return a.statsChannels(r)
		}
	}
	return nil, false, 0, ""
}

func (a *Server) listUsageLogs(r *http.Request) (any, bool, int, string) {
	query := r.URL.Query()
	page, pageSize := ParsePagination(r)

	filter := domain.UsageLogFilter{
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
			return nil, true, http.StatusBadRequest, "invalid user_id"
		}
		filter.UserID = &id
	}
	if value := query.Get("channel_id"); value != "" {
		id, err := strconv.Atoi(value)
		if err != nil {
			return nil, true, http.StatusBadRequest, "invalid channel_id"
		}
		filter.ChannelID = &id
	}
	return a.result(a.store.ListUsageLogs(filter))
}

func (a *Server) statsOverview(r *http.Request) (any, bool, int, string) {
	query := r.URL.Query()
	return a.result(a.store.StatsOverview(orDefault(query.Get("start_time"), defaultStartTime), orDefault(query.Get("end_time"), defaultEndTime)))
}

func (a *Server) statsDaily(r *http.Request) (any, bool, int, string) {
	query := r.URL.Query()
	page, pageSize := ParsePagination(r)
	return a.result(a.store.StatsDaily(orDefault(query.Get("date_from"), defaultDateFrom), orDefault(query.Get("date_to"), defaultDateTo), page, pageSize))
}

func (a *Server) statsChannels(r *http.Request) (any, bool, int, string) {
	query := r.URL.Query()
	return a.result(a.store.StatsChannels(orDefault(query.Get("start_time"), defaultStartTime), orDefault(query.Get("end_time"), defaultEndTime)))
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
