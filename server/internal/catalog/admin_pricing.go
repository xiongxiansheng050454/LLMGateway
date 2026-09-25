package catalog

import (
	"io"
	"net/http"
	"strconv"

	"LLMGateway/server/internal/httpcommon"
)

// RegisterAdminRoutes registers catalog management routes on mux.
func (a *Server) RegisterAdminRoutes(mux *http.ServeMux) {
	httpcommon.HandleAdmin(mux, "/admin/models", a.models)
	httpcommon.HandleAdmin(mux, "/admin/channels", a.channels)
	httpcommon.HandleAdmin(mux, "/admin/channels/{id}", a.channel)
	httpcommon.HandleAdmin(mux, "/admin/channels/{id}/status", a.channelStatus)
	httpcommon.HandleAdmin(mux, "/admin/channels/{id}/balance", a.channelBalance)
	httpcommon.HandleAdmin(mux, "/admin/channels/{id}/health", a.channelHealth)
	httpcommon.HandleAdmin(mux, "/admin/channels/{id}/health/reset", a.channelHealthReset)
	httpcommon.HandleAdmin(mux, "/admin/channels/{id}/test", a.channelTest)
	httpcommon.HandleAdmin(mux, "/admin/channels/{id}/remote-models", a.channelRemoteModels)
	httpcommon.HandleAdmin(mux, "/admin/channels/{id}/models", a.channelModels)
	httpcommon.HandleAdmin(mux, "/admin/channels/{id}/models/{model_id}", a.channelModel)
	httpcommon.HandleAdmin(mux, "/admin/channels/health", a.channelHealthList)
	httpcommon.HandleAdmin(mux, "/admin/pricing", a.pricingData)
}

func (a *Server) models(r *http.Request) httpcommon.AdminResult {
	if r.Method != http.MethodGet {
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
	return httpcommon.Result(a.ListCatalogModels(r.URL.Query().Get("status") == "1"))
}

func (a *Server) channels(r *http.Request) httpcommon.AdminResult {
	switch r.Method {
	case http.MethodGet:
		return httpcommon.Result(a.ListChannels())
	case http.MethodPost:
		return a.createChannel(r)
	default:
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *Server) channel(r *http.Request) httpcommon.AdminResult {
	id, result := parseID(r.PathValue("id"), "channel")
	if result.Status != 0 {
		return result
	}
	switch r.Method {
	case http.MethodPut:
		return a.updateChannel(r, id)
	case http.MethodDelete:
		return httpcommon.NoBody(a.DeleteChannel(id))
	default:
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *Server) channelStatus(r *http.Request) httpcommon.AdminResult {
	if r.Method != http.MethodPut {
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
	id, result := parseID(r.PathValue("id"), "channel")
	if result.Status != 0 {
		return result
	}
	return a.updateChannelStatus(r, id)
}

func (a *Server) channelBalance(r *http.Request) httpcommon.AdminResult {
	if r.Method != http.MethodPut {
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
	id, result := parseID(r.PathValue("id"), "channel")
	if result.Status != 0 {
		return result
	}
	return a.updateChannelBalance(r, id)
}

func (a *Server) channelHealth(r *http.Request) httpcommon.AdminResult {
	if r.Method != http.MethodGet {
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
	id, result := parseID(r.PathValue("id"), "channel")
	if result.Status != 0 {
		return result
	}
	if _, err := a.GetChannelSecret(id); err != nil {
		return httpcommon.Result(nil, err)
	}
	health, err := a.GetChannelHealth(id)
	if err != nil {
		return httpcommon.Result(nil, err)
	}
	return httpcommon.Result(ChannelHealthDTO{ChannelID: health.ChannelID, State: string(health.State), ConsecutiveFailures: health.ConsecutiveFailures, SuccessCount: health.SuccessCount, FailureCount: health.FailureCount, OpenedAt: health.OpenedAt, UpdatedAt: health.UpdatedAt}, nil)
}

func (a *Server) channelHealthReset(r *http.Request) httpcommon.AdminResult {
	if r.Method != http.MethodPost {
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
	id, result := parseID(r.PathValue("id"), "channel")
	if result.Status != 0 {
		return result
	}
	if _, err := a.GetChannelSecret(id); err != nil {
		return httpcommon.NoBody(err)
	}
	return httpcommon.NoBody(a.ResetChannelHealth(id))
}

func (a *Server) channelTest(r *http.Request) httpcommon.AdminResult {
	if r.Method != http.MethodPost {
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
	id, result := parseID(r.PathValue("id"), "channel")
	if result.Status != 0 {
		return result
	}
	return a.testChannel(r, id)
}

func (a *Server) channelRemoteModels(r *http.Request) httpcommon.AdminResult {
	if r.Method != http.MethodPost {
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
	id, result := parseID(r.PathValue("id"), "channel")
	if result.Status != 0 {
		return result
	}
	return a.remoteModels(id)
}

func (a *Server) channelModels(r *http.Request) httpcommon.AdminResult {
	id, result := parseID(r.PathValue("id"), "channel")
	if result.Status != 0 {
		return result
	}
	if r.Method == http.MethodGet {
		return httpcommon.Result(a.ListChannelModels(id))
	}
	if r.Method == http.MethodPost {
		var req ChannelModel
		if err := httpcommon.ReadJSON(r, &req); err != nil {
			return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
		}
		return httpcommon.Result(a.CreateChannelModel(id, req))
	}
	return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
}

func (a *Server) channelModel(r *http.Request) httpcommon.AdminResult {
	id, result := parseID(r.PathValue("id"), "channel")
	if result.Status != 0 {
		return result
	}
	modelID, result := parseID(r.PathValue("model_id"), "model")
	if result.Status != 0 {
		return result
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
		return httpcommon.Result(a.UpdateChannelModel(id, modelID, req.UpstreamModel, req.Enabled))
	case http.MethodDelete:
		return httpcommon.NoBody(a.DeleteChannelModel(id, modelID))
	default:
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *Server) channelHealthList(r *http.Request) httpcommon.AdminResult {
	if r.Method != http.MethodGet {
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
	return httpcommon.Result(a.ListChannelHealth())
}

func parseID(value, name string) (int, httpcommon.AdminResult) {
	id, err := strconv.Atoi(value)
	if err != nil {
		return 0, httpcommon.HTTPError(http.StatusBadRequest, "invalid "+name+" id")
	}
	return id, httpcommon.AdminResult{}
}

func (a *Server) pricingData(r *http.Request) httpcommon.AdminResult {
	switch r.Method {
	case http.MethodGet:
		return httpcommon.Result(a.ListPricing())
	case http.MethodPost:
		var req PricingInput
		if err := httpcommon.ReadJSON(r, &req); err != nil {
			return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
		}
		return httpcommon.Result(a.UpsertPricing(req))
	case http.MethodDelete:
		var req DeletePricingInput
		if err := httpcommon.ReadJSON(r, &req); err != nil && err != io.EOF {
			return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
		}
		return httpcommon.NoBody(a.DeletePricing(req))
	default:
		return httpcommon.HTTPError(http.StatusMethodNotAllowed, "method not allowed")
	}
}
