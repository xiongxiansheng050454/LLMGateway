package postgres

import (
	"context"

	"LLMGateway/server/internal/db/sqlc"
	domain "LLMGateway/server/internal/ratelimit"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Store) ListRateLimits(ctx context.Context, enabled *bool, page, pageSize int) (domain.ListResponse[domain.RateLimitRuleDTO], error) {
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
		list = append(list, domain.RateLimitRuleToDTO(rateLimitRule(row.ID, row.RuleName, row.TargetType, row.TargetValue, row.Metric, row.LimitValue, row.WindowSeconds, row.Action, row.Priority, row.Enabled, row.Extras)))
	}
	return domain.ListResponse[domain.RateLimitRuleDTO]{List: list, Total: int(total)}, nil
}

func (s *Store) GetRateLimit(ctx context.Context, id int) (domain.RateLimitRule, error) {
	row, err := s.queries.GetRateLimitRule(ctx, int64(id))
	if err != nil {
		return domain.RateLimitRule{}, mapError(err)
	}
	return rateLimitRule(row.ID, row.RuleName, row.TargetType, row.TargetValue, row.Metric, row.LimitValue, row.WindowSeconds, row.Action, row.Priority, row.Enabled, row.Extras), nil
}

func (s *Store) InsertRateLimit(ctx context.Context, rule domain.RateLimitRule) (int, error) {
	id, err := s.queries.CreateRateLimitRule(ctx, sqlc.CreateRateLimitRuleParams{
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
		return 0, mapError(err)
	}
	return int(id), nil
}

func (s *Store) UpdateRateLimitRecord(ctx context.Context, id int, rule domain.RateLimitRule) (bool, error) {
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
		return false, mapError(err)
	}
	return affected > 0, nil
}

func (s *Store) DeleteRateLimit(ctx context.Context, id int) (bool, error) {
	affected, err := s.queries.DeleteRateLimitRule(ctx, int64(id))
	if err != nil {
		return false, mapError(err)
	}
	return affected > 0, nil
}

func rateLimitRule(id int64, ruleName, targetType, targetValue, metric string, limitValue int64, windowSeconds int32, action string, priority int32, enabled bool, extras []byte) domain.RateLimitRule {
	return domain.RateLimitRule{ID: int(id), RuleName: ruleName, TargetType: targetType, TargetValue: targetValue, Metric: metric, LimitValue: limitValue, WindowSeconds: int(windowSeconds), Action: action, Priority: int(priority), Enabled: enabled, Extras: extras}
}
