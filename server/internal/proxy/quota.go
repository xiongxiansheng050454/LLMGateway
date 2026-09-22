package proxy

import (
	"context"
	"errors"
	"fmt"

	"LLMGateway/server/internal/accounts"
	apperrors "LLMGateway/server/internal/errors"
	"LLMGateway/server/internal/money"
	"LLMGateway/server/internal/quota"
)

func (a *Service) reserveQuota(ctx context.Context, requestID string, auth *accounts.AuthContext, req ChatRequest, channelID int) (quota.QuotaReservation, error) {
	if a.adapter.EstimateUsage == nil {
		return quota.QuotaReservation{}, ErrInvalidRequest
	}
	estimate, err := a.adapter.EstimateUsage(req.Body, a.defaultMaxTokens)
	if err != nil {
		return quota.QuotaReservation{}, ErrInvalidRequest
	}
	estimatedCost, err := a.estimatedCost(channelID, req.Model, estimate)
	if err != nil {
		return quota.QuotaReservation{}, err
	}
	reservation, err := a.store.ReserveQuota(ctx, quota.QuotaReserveInput{
		RequestID: requestID, UserID: auth.UserID, APIKeyID: auth.KeyID, Model: req.Model,
		EstimatedTokens: int64(estimate.TotalTokens), EstimatedCost: estimatedCost,
		ExpiresAt: a.now().Add(a.reservationTTL),
	})
	if errors.Is(err, apperrors.ErrQuotaExceeded) {
		return quota.QuotaReservation{}, ErrQuotaExceeded
	}
	return reservation, err
}

func (a *Service) estimatedCost(channelID int, model string, estimate EstimatedUsage) (string, error) {
	pricing, err := a.catalog.GetPricing(channelID, model)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
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
		return "", fmt.Errorf("%w: invalid estimated cost", apperrors.ErrInvalid)
	}
	return money.Format6(money.Amount(total)), nil
}
