package storefake

import (
	"sort"

	domain "LLMGateway/server/internal/ratelimit"
	"LLMGateway/server/internal/store"
)

func (s *Store) ListRateLimits(enabled *bool, page, pageSize int) (domain.ListResponse[domain.RateLimitRuleDTO], error) {
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
	list := []domain.RateLimitRuleDTO{}
	for _, rule := range rules[start:end] {
		list = append(list, domain.RateLimitRuleToDTO(*rule))
	}
	return domain.ListResponse[domain.RateLimitRuleDTO]{List: list, Total: len(rules)}, nil
}

func (s *Store) GetRateLimit(id int) (domain.RateLimitRule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rule, ok := s.rateLimits[id]
	if !ok {
		return domain.RateLimitRule{}, store.ErrNotFound
	}
	return *rule, nil
}

func (s *Store) InsertRateLimit(rule domain.RateLimitRule) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rule.ID = s.nextRateLimitID
	s.nextRateLimitID++
	stored := rule
	s.rateLimits[stored.ID] = &stored
	return stored.ID, nil
}

func (s *Store) UpdateRateLimitRecord(id int, rule domain.RateLimitRule) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rateLimits[id]; !ok {
		return false, nil
	}
	rule.ID = id
	stored := rule
	s.rateLimits[id] = &stored
	return true, nil
}

func (s *Store) DeleteRateLimit(id int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rateLimits[id]; !ok {
		return false, nil
	}
	delete(s.rateLimits, id)
	return true, nil
}
