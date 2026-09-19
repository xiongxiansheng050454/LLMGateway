package postgres

import (
	"context"

	"LLMGateway/server/internal/db/sqlc"
	domain "LLMGateway/server/internal/ratelimit"
	"LLMGateway/server/internal/store"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Store) ListRateLimits(enabled *bool, page, pageSize int) (domain.ListResponse[domain.RateLimitRuleDTO], error) {
	ctx := context.Background()
	limit, offset := limitOffset(page, pageSize)

	var enabledArg pgtype.Bool
	if enabled != nil {
		enabledArg = pgtype.Bool{Bool: *enabled, Valid: true}
	}

	rows, err := s.queries.ListRateLimitRules(ctx, sqlc.ListRateLimitRulesParams{PageLimit: limit, PageOffset: offset, Enabled: enabledArg})
	if err != nil {
		return domain.ListResponse[domain.RateLimitRuleDTO]{}, mapError(err)
	}
	total, err := s.queries.CountRateLimitRules(ctx, enabledArg)
	if err != nil {
		return domain.ListResponse[domain.RateLimitRuleDTO]{}, mapError(err)
	}

	list := []domain.RateLimitRuleDTO{}
	for _, row := range rows {
		list = append(list, rateLimitDTO(row.ID, row.RuleName, row.TargetType, row.TargetValue, row.Metric, row.LimitValue, row.WindowSeconds, row.Action, row.Priority, row.Enabled, row.Extras))
	}
	return domain.ListResponse[domain.RateLimitRuleDTO]{List: list, Total: int(total)}, nil
}

func (s *Store) CreateRateLimit(in domain.RateLimitInput) (domain.RateLimitRuleDTO, error) {
	rule, err := domain.NormalizeRateLimit(in, nil)
	if err != nil {
		return domain.RateLimitRuleDTO{}, err
	}

	id, err := s.queries.CreateRateLimitRule(context.Background(), sqlc.CreateRateLimitRuleParams{
		RuleName:      rule.RuleName,
		TargetType:    rule.TargetType,
		TargetValue:   rule.TargetValue,
		Metric:        rule.Metric,
		LimitValue:    rule.LimitValue,
		WindowSeconds: int32(rule.WindowSeconds),
		Action:        rule.Action,
		Priority:      int32(rule.Priority),
		Enabled:       rule.Enabled,
		Extras:        rule.Extras,
	})
	if err != nil {
		return domain.RateLimitRuleDTO{}, mapError(err)
	}
	return s.getRateLimitDTO(id)
}

func (s *Store) UpdateRateLimit(id int, in domain.RateLimitInput) (domain.RateLimitRuleDTO, error) {
	ctx := context.Background()
	current, err := s.queries.GetRateLimitRule(ctx, int64(id))
	if err != nil {
		return domain.RateLimitRuleDTO{}, mapError(err)
	}

	existing := &domain.RateLimitRule{
		ID:            int(current.ID),
		RuleName:      current.RuleName,
		TargetType:    current.TargetType,
		TargetValue:   current.TargetValue,
		Metric:        current.Metric,
		LimitValue:    current.LimitValue,
		WindowSeconds: int(current.WindowSeconds),
		Action:        current.Action,
		Priority:      int(current.Priority),
		Enabled:       current.Enabled,
		Extras:        current.Extras,
	}
	rule, err := domain.NormalizeRateLimit(in, existing)
	if err != nil {
		return domain.RateLimitRuleDTO{}, err
	}

	affected, err := s.queries.UpdateRateLimitRule(ctx, sqlc.UpdateRateLimitRuleParams{
		RuleName:      rule.RuleName,
		TargetType:    rule.TargetType,
		TargetValue:   rule.TargetValue,
		Metric:        rule.Metric,
		LimitValue:    rule.LimitValue,
		WindowSeconds: int32(rule.WindowSeconds),
		Action:        rule.Action,
		Priority:      int32(rule.Priority),
		Enabled:       rule.Enabled,
		Extras:        rule.Extras,
		ID:            int64(id),
	})
	if err != nil {
		return domain.RateLimitRuleDTO{}, mapError(err)
	}
	if affected == 0 {
		return domain.RateLimitRuleDTO{}, store.ErrNotFound
	}
	return s.getRateLimitDTO(int64(id))
}

func (s *Store) DeleteRateLimit(id int) error {
	affected, err := s.queries.DeleteRateLimitRule(context.Background(), int64(id))
	if err != nil {
		return mapError(err)
	}
	if affected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *Store) getRateLimitDTO(id int64) (domain.RateLimitRuleDTO, error) {
	row, err := s.queries.GetRateLimitRule(context.Background(), id)
	if err != nil {
		return domain.RateLimitRuleDTO{}, mapError(err)
	}
	return rateLimitDTO(row.ID, row.RuleName, row.TargetType, row.TargetValue, row.Metric, row.LimitValue, row.WindowSeconds, row.Action, row.Priority, row.Enabled, row.Extras), nil
}

func rateLimitDTO(id int64, ruleName, targetType, targetValue, metric string, limitValue int64, windowSeconds int32, action string, priority int32, enabled bool, extras []byte) domain.RateLimitRuleDTO {
	return domain.RateLimitRuleDTO{ID: int(id), RuleName: ruleName, TargetType: targetType, TargetValue: targetValue, Metric: metric, LimitValue: limitValue, WindowSeconds: int(windowSeconds), Action: action, Priority: int(priority), Enabled: enabled, Extras: extras}
}
