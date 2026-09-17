package memory

import (
	"sort"

	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/store"
)

func (s *Store) ListRateLimits(enabled *bool, page, pageSize int) (domain.ListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rules := []*domain.RateLimitRule{}
	for _, rule := range s.rateLimits {
		if enabled != nil && rule.Enabled != *enabled {
			continue
		}
		rules = append(rules, rule)
	}
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Priority != rules[j].Priority {
			return rules[i].Priority < rules[j].Priority
		}
		return rules[i].ID < rules[j].ID
	})

	start, end := pageBounds(len(rules), page, pageSize)
	list := []any{}
	for _, rule := range rules[start:end] {
		list = append(list, rateLimitDTO(rule))
	}
	return domain.ListResponse{List: list, Total: len(rules)}, nil
}

func (s *Store) CreateRateLimit(in domain.RateLimitInput) (map[string]any, error) {
	rule, err := domain.NormalizeRateLimit(in, nil)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	rule.ID = s.nextRateLimitID
	s.nextRateLimitID++
	stored := rule
	s.rateLimits[stored.ID] = &stored
	return rateLimitDTO(&stored), nil
}

func (s *Store) UpdateRateLimit(id int, in domain.RateLimitInput) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.rateLimits[id]
	if !ok {
		return nil, store.ErrNotFound
	}

	rule, err := domain.NormalizeRateLimit(in, existing)
	if err != nil {
		return nil, err
	}
	rule.ID = id
	*s.rateLimits[id] = rule
	return rateLimitDTO(&rule), nil
}

func (s *Store) DeleteRateLimit(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rateLimits[id]; !ok {
		return store.ErrNotFound
	}
	delete(s.rateLimits, id)
	return nil
}

func rateLimitDTO(rule *domain.RateLimitRule) map[string]any {
	return map[string]any{
		"id":             rule.ID,
		"rule_name":      rule.RuleName,
		"target_type":    rule.TargetType,
		"target_value":   rule.TargetValue,
		"metric":         rule.Metric,
		"limit_value":    rule.LimitValue,
		"window_seconds": rule.WindowSeconds,
		"action":         rule.Action,
		"priority":       rule.Priority,
		"enabled":        rule.Enabled,
		"extras":         rule.Extras,
	}
}
