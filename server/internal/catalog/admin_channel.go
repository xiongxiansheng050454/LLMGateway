package catalog

import (
	"net/http"

	"LLMGateway/server/internal/httpcommon"
)

func (a *Server) createChannel(r *http.Request) httpcommon.AdminResult {
	var req ChannelInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.CreateChannel(r.Context(), req))
}

func (a *Server) updateChannel(r *http.Request, id int) httpcommon.AdminResult {
	var req ChannelInput
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.UpdateChannel(r.Context(), id, req))
}

func (a *Server) updateChannelStatus(r *http.Request, id int) httpcommon.AdminResult {
	var req struct {
		Status int `json:"status"`
	}
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.UpdateChannelStatus(r.Context(), id, req.Status))
}

func (a *Server) updateChannelBalance(r *http.Request, id int) httpcommon.AdminResult {
	var req struct {
		Balance string `json:"balance"`
		Delta   string `json:"delta"`
	}
	if err := httpcommon.ReadJSON(r, &req); err != nil {
		return httpcommon.HTTPError(http.StatusBadRequest, "invalid json")
	}
	return httpcommon.Result(a.UpdateChannelBalance(r.Context(), id, req.Balance, req.Delta))
}
