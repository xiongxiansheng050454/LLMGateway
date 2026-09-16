package handler

import (
	"io"
	"net/http"

	"LLMGateway/server/internal/domain"
)

func (a *Server) pricingData(r *http.Request) (any, bool, int, string) {
	switch r.Method {
	case http.MethodGet:
		return a.result(a.store.ListPricing())
	case http.MethodPost:
		var req domain.PricingInput
		if err := readJSON(r, &req); err != nil {
			return nil, true, http.StatusBadRequest, "invalid json"
		}
		return a.result(a.store.UpsertPricing(req))
	case http.MethodDelete:
		var req domain.DeletePricingInput
		if err := readJSON(r, &req); err != nil && err != io.EOF {
			return nil, true, http.StatusBadRequest, "invalid json"
		}
		return a.noBody(a.store.DeletePricing(req))
	default:
		return nil, true, http.StatusMethodNotAllowed, "method not allowed"
	}
}
