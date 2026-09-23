package catalog

import (
	"net/http"
	"strconv"

	"LLMGateway/server/internal/httpcommon"
)

func (a *Server) channelData(r *http.Request, parts []string) httpcommon.AdminResult {
	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			return httpcommon.Result(a.ListChannels())
		case http.MethodPost:
			return a.createChannel(r)
		}
	}
	if len(parts) < 3 {
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
	channelID, err := strconv.Atoi(parts[2])
	if err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid channel id")
	}
	if len(parts) == 3 {
		switch r.Method {
		case http.MethodPut:
			return a.updateChannel(r, channelID)
		case http.MethodDelete:
			return httpcommon.NoBody(a.DeleteChannel(channelID))
		}
	}
	if len(parts) == 4 && parts[3] == "status" && r.Method == http.MethodPut {
		return a.updateChannelStatus(r, channelID)
	}
	if len(parts) == 4 && parts[3] == "balance" && r.Method == http.MethodPut {
		return a.updateChannelBalance(r, channelID)
	}
	if len(parts) == 4 && parts[3] == "health" && r.Method == http.MethodGet {
		if _, err := a.GetChannelSecret(channelID); err != nil {
			return httpcommon.Result(nil, err)
		}
		health, err := a.GetChannelHealth(channelID)
		if err != nil {
			return httpcommon.Result(nil, err)
		}
		return httpcommon.Result(ChannelHealthDTO{
			ChannelID:           health.ChannelID,
			State:               string(health.State),
			ConsecutiveFailures: health.ConsecutiveFailures,
			SuccessCount:        health.SuccessCount,
			FailureCount:        health.FailureCount,
			OpenedAt:            health.OpenedAt,
			UpdatedAt:           health.UpdatedAt,
		}, nil)
	}
	if len(parts) == 5 && parts[3] == "health" && parts[4] == "reset" && r.Method == http.MethodPost {
		if _, err := a.GetChannelSecret(channelID); err != nil {
			return httpcommon.NoBody(err)
		}
		return httpcommon.NoBody(a.ResetChannelHealth(channelID))
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
	return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
}

func (a *Server) channelModelData(r *http.Request, parts []string, channelID int) httpcommon.AdminResult {
	if len(parts) == 4 {
		switch r.Method {
		case http.MethodGet:
			return httpcommon.Result(a.ListChannelModels(channelID))
		case http.MethodPost:
			var req ChannelModel
			if err := httpcommon.ReadJSON(r, &req); err != nil {
				return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
			}
			return httpcommon.Result(a.CreateChannelModel(channelID, req))
		}
	}
	if len(parts) == 5 {
		modelID, err := strconv.Atoi(parts[4])
		if err != nil {
			return httpcommon.HTTPError(http.StatusBadRequest, "invalid model id")
		}
		switch r.Method {
		case http.MethodPut:
			var req struct {
				UpstreamModel string `json:"upstream_model"`
				Enabled       bool   `json:"enabled"`
			}
			if err := httpcommon.ReadJSON(r, &req); err != nil {
				return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
			}
			return httpcommon.Result(a.UpdateChannelModel(channelID, modelID, req.UpstreamModel, req.Enabled))
		case http.MethodDelete:
			return httpcommon.NoBody(a.DeleteChannelModel(channelID, modelID))
		}
	}
	return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
}

func (a *Server) createChannel(r *http.Request) httpcommon.AdminResult {
	var req ChannelInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.CreateChannel(req))
}

func (a *Server) updateChannel(r *http.Request, id int) httpcommon.AdminResult {
	var req ChannelInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.UpdateChannel(id, req))
}

func (a *Server) updateChannelStatus(r *http.Request, id int) httpcommon.AdminResult {
	var req struct {
		Status int `json:"status"`
	}
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.UpdateChannelStatus(id, req.Status))
}

func (a *Server) updateChannelBalance(r *http.Request, id int) httpcommon.AdminResult {
	var req struct {
		Balance string `json:"balance"`
		Delta   string `json:"delta"`
	}
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.UpdateChannelBalance(id, req.Balance, req.Delta))
}
