package catalog

import (
	"net/http"
	"strconv"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/httpcommon"
)

func (a *Server) channelData(r *http.Request, parts []string) (any, bool, int, string) {
	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			return httpcommon.Result(a.store.ListChannels())
		case http.MethodPost:
			return a.createChannel(r)
		}
	}
	if len(parts) < 3 {
		return nil, true, http.StatusMethodNotAllowed, "method not allowed"
	}
	channelID, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, true, http.StatusBadRequest, "invalid channel id"
	}
	if len(parts) == 3 {
		switch r.Method {
		case http.MethodPut:
			return a.updateChannel(r, channelID)
		case http.MethodDelete:
			return httpcommon.NoBody(a.store.DeleteChannel(channelID))
		}
	}
	if len(parts) == 4 && parts[3] == "status" && r.Method == http.MethodPut {
		return a.updateChannelStatus(r, channelID)
	}
	if len(parts) == 4 && parts[3] == "balance" && r.Method == http.MethodPut {
		return a.updateChannelBalance(r, channelID)
	}
	if len(parts) == 4 && parts[3] == "health" && r.Method == http.MethodGet {
		if _, err := a.store.GetChannelSecret(channelID); err != nil {
			return httpcommon.Result(nil, err)
		}
		health, err := a.store.GetChannelHealth(channelID)
		if err != nil {
			return httpcommon.Result(nil, err)
		}
		return domain.ChannelHealthDTO{ChannelID: health.ChannelID, State: string(health.State), ConsecutiveFailures: health.ConsecutiveFailures, SuccessCount: health.SuccessCount, FailureCount: health.FailureCount, OpenedAt: health.OpenedAt, UpdatedAt: health.UpdatedAt}, true, 0, ""
	}
	if len(parts) == 5 && parts[3] == "health" && parts[4] == "reset" && r.Method == http.MethodPost {
		if _, err := a.store.GetChannelSecret(channelID); err != nil {
			return httpcommon.NoBody(err)
		}
		return httpcommon.NoBody(a.store.ResetChannelHealth(channelID))
	}
	if len(parts) == 4 && parts[3] == "test" && r.Method == http.MethodPost {
		return a.testChannel(r, channelID)
	}
	if len(parts) == 4 && parts[3] == "remote-models" && r.Method == http.MethodPost {
		return a.remoteModels(channelID)
	}
	if len(parts) >= 4 && parts[3] == "models" {
		return a.channelModelData(r, parts, channelID)
	}
	return nil, true, http.StatusMethodNotAllowed, "method not allowed"
}

func (a *Server) channelModelData(r *http.Request, parts []string, channelID int) (any, bool, int, string) {
	if len(parts) == 4 {
		switch r.Method {
		case http.MethodGet:
			return httpcommon.Result(a.store.ListChannelModels(channelID))
		case http.MethodPost:
			var req domain.ChannelModel
			if err := httpcommon.ReadJSON(r, &req); err != nil {
				return nil, true, http.StatusBadRequest, "invalid json"
			}
			return httpcommon.Result(a.store.CreateChannelModel(channelID, req))
		}
	}
	if len(parts) == 5 {
		modelID, err := strconv.Atoi(parts[4])
		if err != nil {
			return nil, true, http.StatusBadRequest, "invalid model id"
		}
		switch r.Method {
		case http.MethodPut:
			var req struct {
				UpstreamModel string `json:"upstream_model"`
				Enabled       bool   `json:"enabled"`
			}
			if err := httpcommon.ReadJSON(r, &req); err != nil {
				return nil, true, http.StatusBadRequest, "invalid json"
			}
			return httpcommon.Result(a.store.UpdateChannelModel(channelID, modelID, req.UpstreamModel, req.Enabled))
		case http.MethodDelete:
			return httpcommon.NoBody(a.store.DeleteChannelModel(channelID, modelID))
		}
	}
	return nil, true, http.StatusMethodNotAllowed, "method not allowed"
}

func (a *Server) createChannel(r *http.Request) (any, bool, int, string) {
	var req domain.ChannelInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return httpcommon.Result(a.store.CreateChannel(req))
}

func (a *Server) updateChannel(r *http.Request, id int) (any, bool, int, string) {
	var req domain.ChannelInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return httpcommon.Result(a.store.UpdateChannel(id, req))
}

func (a *Server) updateChannelStatus(r *http.Request, id int) (any, bool, int, string) {
	var req struct {
		Status int `json:"status"`
	}
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return httpcommon.Result(a.store.UpdateChannelStatus(id, req.Status))
}

func (a *Server) updateChannelBalance(r *http.Request, id int) (any, bool, int, string) {
	var req struct {
		Balance string `json:"balance"`
		Delta   string `json:"delta"`
	}
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return httpcommon.Result(a.store.UpdateChannelBalance(id, req.Balance, req.Delta))
}
