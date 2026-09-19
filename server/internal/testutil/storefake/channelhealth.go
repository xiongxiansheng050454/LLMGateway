package storefake

import (
	"context"
	"sort"
	"time"

	domain "LLMGateway/server/internal/catalog"
)

func (s *Store) RecordChannelAttempt(_ context.Context, channelID int, success bool, reason domain.FailureReason) (domain.ChannelHealth, error) {
	if !success && !reason.CountsAsChannelFailure() {
		return s.GetChannelHealth(channelID)
	}
	if success {
		return s.RecordChannelSuccess(channelID)
	}
	return s.RecordChannelFailure(channelID, reason)
}

func (s *Store) AcquireChannelProbe(_ context.Context, channelID int, lease time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if until, ok := s.probes[channelID]; ok && until.After(now) {
		return false, nil
	}
	s.probes[channelID] = now.Add(lease)
	return true, nil
}

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
	delete(s.probes, channelID)
	return nil
}

func (s *Store) ListChannelHealth() (domain.ListResponse[domain.ChannelHealthDTO], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]int, 0, len(s.channels))
	for id := range s.channels {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	list := []domain.ChannelHealthDTO{}
	for _, id := range ids {
		health := s.channelHealthLocked(id)
		list = append(list, channelHealthDTO(&health))
	}
	return domain.ListResponse[domain.ChannelHealthDTO]{List: list, Total: len(ids)}, nil
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

func channelHealthDTO(health *domain.ChannelHealth) domain.ChannelHealthDTO {
	return domain.ChannelHealthDTO{ChannelID: health.ChannelID, State: string(health.State), ConsecutiveFailures: health.ConsecutiveFailures, SuccessCount: health.SuccessCount, FailureCount: health.FailureCount, OpenedAt: health.OpenedAt, UpdatedAt: health.UpdatedAt}
}
