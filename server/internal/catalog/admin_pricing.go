package catalog

import (
	"io"
	"net/http"

	"LLMGateway/server/internal/httpcommon"
)

func (a *Server) Data(r *http.Request, parts []string) httpcommon.AdminResult {
	if len(parts) == 2 && parts[1] == "models" && r.Method == http.MethodGet {
		return httpcommon.Result(a.ListCatalogModels(r.URL.Query().Get("status") == "1"))
	}
	if len(parts) >= 2 && parts[1] == "channels" {
		if len(parts) == 3 && parts[2] == "health" && r.Method == http.MethodGet {
			return httpcommon.Result(a.ListChannelHealth())
		}
		return a.channelData(r, parts)
	}
	if len(parts) == 2 && parts[1] == "pricing" {
		return a.pricingData(r)
	}
	return httpcommon.Unhandled()
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
