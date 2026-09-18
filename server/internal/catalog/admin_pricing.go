package catalog

import (
	"io"
	"net/http"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/httpcommon"
)

func (a *Server) Data(r *http.Request, parts []string) (any, bool, int, string) {
	if len(parts) == 2 && parts[1] == "models" && r.Method == http.MethodGet {
		return httpcommon.Result(a.store.ListCatalogModels(r.URL.Query().Get("status") == "1"))
	}
	if len(parts) >= 2 && parts[1] == "channels" {
		if len(parts) == 3 && parts[2] == "health" && r.Method == http.MethodGet {
			return httpcommon.Result(a.store.ListChannelHealth())
		}
		return a.channelData(r, parts)
	}
	if len(parts) == 2 && parts[1] == "pricing" {
		return a.pricingData(r)
	}
	return nil, false, 0, ""
}

func (a *Server) pricingData(r *http.Request) (any, bool, int, string) {
	switch r.Method {
	case http.MethodGet:
		return httpcommon.Result(a.store.ListPricing())
	case http.MethodPost:
		var req domain.PricingInput
		if err := httpcommon.ReadJSON(r, &req); err != nil {
			return nil, true, http.StatusBadRequest, "invalid json"
		}
		return httpcommon.Result(a.store.UpsertPricing(req))
	case http.MethodDelete:
		var req domain.DeletePricingInput
		if err := httpcommon.ReadJSON(r, &req); err != nil && err != io.EOF {
			return nil, true, http.StatusBadRequest, "invalid json"
		}
		return httpcommon.NoBody(a.store.DeletePricing(req))
	default:
		return nil, true, http.StatusMethodNotAllowed, "method not allowed"
	}
}
