package ratelimit

import "context"

func (a *Server) ListRateLimits(ctx context.Context, enabled *bool, page, pageSize int) (ListResponse[RateLimitRuleDTO], error) {
	return a.store.ListRateLimits(ctx, enabled, page, pageSize)
}

// CreateRateLimit normalizes and persists a new rule.
func (a *Server) CreateRateLimit(ctx context.Context, in RateLimitInput) (RateLimitRuleDTO, error) {
	rule, err := NormalizeRateLimit(in, nil)
	if err != nil {
		return RateLimitRuleDTO{}, err
	}
	id, err := a.store.InsertRateLimit(ctx, rule)
	if err != nil {
		return RateLimitRuleDTO{}, err
	}
	rule.ID = id
	return RateLimitRuleToDTO(rule), nil
}

// UpdateRateLimit merges the partial input over the stored rule and persists it.
func (a *Server) UpdateRateLimit(ctx context.Context, id int, in RateLimitInput) (RateLimitRuleDTO, error) {
	existing, err := a.store.GetRateLimit(ctx, id)
	if err != nil {
		return RateLimitRuleDTO{}, err
	}
	rule, err := NormalizeRateLimit(in, &existing)
	if err != nil {
		return RateLimitRuleDTO{}, err
	}
	ok, err := a.store.UpdateRateLimitRecord(ctx, id, rule)
	if err != nil {
		return RateLimitRuleDTO{}, err
	}
	if !ok {
		return RateLimitRuleDTO{}, ErrNotFound
	}
	rule.ID = id
	return RateLimitRuleToDTO(rule), nil
}

func (a *Server) DeleteRateLimit(ctx context.Context, id int) error {
	ok, err := a.store.DeleteRateLimit(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

// ReserveRateLimit validates the reservation request and records it.
func (a *Server) ReserveRateLimit(ctx context.Context, in RateLimitReservationInput) (RateLimitReservation, error) {
	if in.RequestID == "" || in.UserID <= 0 || in.APIKeyID <= 0 || in.EstimatedTokens < 0 || !in.ExpiresAt.After(a.now()) {
		return RateLimitReservation{}, ErrInvalid
	}
	id, err := a.store.InsertRateLimitReservation(ctx, in)
	if err != nil {
		return RateLimitReservation{}, err
	}
	return RateLimitReservation{ID: id}, nil
}

func (a *Server) FinalizeRateLimit(ctx context.Context, id int64, tokens int64) error {
	ok, err := a.store.FinalizeRateLimitReservation(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (a *Server) ReleaseRateLimit(ctx context.Context, id int64) error {
	ok, err := a.store.ReleaseRateLimitReservation(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (a *Server) ReapRateLimitReservations(ctx context.Context, limit int) (int, error) {
	return a.store.ReapRateLimitReservations(ctx, limit)
}

func (a *Server) CountActiveRateLimitReservations(ctx context.Context, userID int, apiKeyID *int, model string, channelID *int) (int64, error) {
	return a.store.CountActiveRateLimitReservations(ctx, userID, apiKeyID, model, channelID)
}

// RateLimitRuleToDTO maps a rule to its wire representation.
func RateLimitRuleToDTO(rule RateLimitRule) RateLimitRuleDTO {
	return RateLimitRuleDTO{
		ID:            rule.ID,
		RuleName:      rule.RuleName,
		TargetType:    rule.TargetType,
		TargetValue:   rule.TargetValue,
		Metric:        rule.Metric,
		LimitValue:    rule.LimitValue,
		WindowSeconds: rule.WindowSeconds,
		Action:        rule.Action,
		Priority:      rule.Priority,
		Enabled:       rule.Enabled,
		Extras:        rule.Extras,
	}
}
