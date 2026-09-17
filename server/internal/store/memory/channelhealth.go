package memory

import (
	"sort"

	"LLMGateway/server/internal/domain"
)

func (s *Store) GetChannelHealth(channelID int) (domain.ChannelHealth, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.channelHealthLocked(channelID), nil
}

func (s *Store) RecordChannelSuccess(channelID int) (domain.ChannelHealth, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := domain.ApplyChannelSuccess(s.channelHealthLocked(channelID), s.now())
	s.channelHealth[channelID] = &next
	return next, nil
}

func (s *Store) RecordChannelFailure(channelID int, reason domain.FailureReason) (domain.ChannelHealth, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := domain.ApplyChannelFailure(s.channelHealthLocked(channelID), reason, s.now(), s.breaker)
	s.channelHealth[channelID] = &next
	return next, nil
}

func (s *Store) ResetChannelHealth(channelID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.channelHealth, channelID)
	return nil
}

func (s *Store) ListChannelHealth() (domain.ListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]int, 0, len(s.channelHealth))
	for id := range s.channelHealth {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	list := []any{}
	for _, id := range ids {
		health := s.channelHealthLocked(id)
		list = append(list, channelHealthDTO(&health))
	}
	return domain.ListResponse{List: list, Total: len(ids)}, nil
}

// channelHealthLocked returns the channel health with the lazy open ->
// half-open transition applied. It does not persist the transition.
func (s *Store) channelHealthLocked(channelID int) domain.ChannelHealth {
	current, ok := s.channelHealth[channelID]
	if !ok {
		return domain.NewChannelHealth(channelID)
	}
	return domain.EvaluateChannelHealth(*current, s.now(), s.breaker)
}

func channelHealthDTO(health *domain.ChannelHealth) map[string]any {
	return map[string]any{
		"channel_id":           health.ChannelID,
		"state":                string(health.State),
		"consecutive_failures": health.ConsecutiveFailures,
		"success_count":        health.SuccessCount,
		"failure_count":        health.FailureCount,
		"opened_at":            health.OpenedAt,
		"updated_at":           health.UpdatedAt,
	}
}
