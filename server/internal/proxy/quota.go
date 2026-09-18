package proxy

import (
	"context"
	"errors"
	"fmt"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/money"
	"LLMGateway/server/internal/store"
)

func (a *Service) reserveQuota(ctx context.Context, requestID string, auth *domain.AuthContext, req ChatRequest, channelID int) (domain.QuotaReservation, error) {
	if a.adapter.EstimateUsage == nil {
		return domain.QuotaReservation{}, ErrInvalidRequest
	}
	estimate, err := a.adapter.EstimateUsage(req.Body, a.defaultMaxTokens)
	if err != nil {
		return domain.QuotaReservation{}, ErrInvalidRequest
	}
	estimatedCost, err := a.estimatedCost(channelID, req.Model, estimate)
	if err != nil {
		return domain.QuotaReservation{}, err
	}
	reservation, err := a.store.ReserveQuota(ctx, domain.QuotaReserveInput{
		RequestID: requestID, UserID: auth.UserID, APIKeyID: auth.KeyID, Model: req.Model,
		EstimatedTokens: int64(estimate.TotalTokens), EstimatedCost: estimatedCost,
		ExpiresAt: a.now().Add(a.reservationTTL),
	})
	if errors.Is(err, store.ErrQuotaExceeded) {
		return domain.QuotaReservation{}, ErrQuotaExceeded
	}
	return reservation, err
}

func (a *Service) estimatedCost(channelID int, model string, estimate EstimatedUsage) (string, error) {
	pricing, err := a.store.GetPricing(channelID, model)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "0.000000", nil
		}
		return "", err
	}
	inputPrice, err := parsePrice8(pricing.InputPricePer1M)
	if err != nil {
		return "", err
	}
	cachedPrice, err := parsePrice8(pricing.CachedInputPricePer1M)
	if err != nil {
		return "", err
	}
	if cachedPrice > inputPrice {
		inputPrice = cachedPrice
	}
	outputPrice, err := parsePrice8(pricing.OutputPricePer1M)
	if err != nil {
		return "", err
	}
	total := priceTokens(inputPrice, estimate.InputTokens) + priceTokens(outputPrice, estimate.OutputTokens)
	if total < 0 {
		return "", fmt.Errorf("%w: invalid estimated cost", store.ErrInvalid)
	}
	return money.Format6(money.Amount(total)), nil
}
