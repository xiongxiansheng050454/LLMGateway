package httpapi

import (
	"net/http"
	"strconv"

	"LLMGateway/server/internal/domain"
)

func (a *Server) channelData(r *http.Request, parts []string) (any, bool, int, string) {
	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			return a.result(a.store.ListChannels())
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
			return a.noBody(a.store.DeleteChannel(channelID))
		}
	}
	if len(parts) == 4 && parts[3] == "status" && r.Method == http.MethodPut {
		return a.updateChannelStatus(r, channelID)
	}
	if len(parts) == 4 && parts[3] == "balance" && r.Method == http.MethodPut {
		return a.updateChannelBalance(r, channelID)
	}
	if len(parts) == 4 && parts[3] == "test" && r.Method == http.MethodPost {
		return a.result(a.store.TestChannel(channelID))
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
			return a.result(a.store.ListChannelModels(channelID))
		case http.MethodPost:
			var req domain.ChannelModel
			if err := readJSON(r, &req); err != nil {
				return nil, true, http.StatusBadRequest, "invalid json"
			}
			return a.result(a.store.CreateChannelModel(channelID, req))
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
			if err := readJSON(r, &req); err != nil {
				return nil, true, http.StatusBadRequest, "invalid json"
			}
			return a.result(a.store.UpdateChannelModel(channelID, modelID, req.UpstreamModel, req.Enabled))
		case http.MethodDelete:
			return a.noBody(a.store.DeleteChannelModel(channelID, modelID))
		}
	}
	return nil, true, http.StatusMethodNotAllowed, "method not allowed"
}

func (a *Server) createChannel(r *http.Request) (any, bool, int, string) {
	var req domain.ChannelInput
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return a.result(a.store.CreateChannel(req))
}

func (a *Server) updateChannel(r *http.Request, id int) (any, bool, int, string) {
	var req domain.ChannelInput
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return a.result(a.store.UpdateChannel(id, req))
}

func (a *Server) updateChannelStatus(r *http.Request, id int) (any, bool, int, string) {
	var req struct {
		Status int `json:"status"`
	}
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return a.result(a.store.UpdateChannelStatus(id, req.Status))
}

func (a *Server) updateChannelBalance(r *http.Request, id int) (any, bool, int, string) {
	var req struct {
		Balance string `json:"balance"`
		Delta   string `json:"delta"`
	}
	if err := readJSON(r, &req); err != nil {
		return nil, true, http.StatusBadRequest, "invalid json"
	}
	return a.result(a.store.UpdateChannelBalance(id, req.Balance, req.Delta))
}
