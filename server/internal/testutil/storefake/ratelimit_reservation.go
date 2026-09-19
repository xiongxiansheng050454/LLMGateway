package storefake

import (
	domain "LLMGateway/server/internal/ratelimit"
	"LLMGateway/server/internal/store"
	"context"
	"time"
)

func (s *Store) ReserveRateLimit(_ context.Context, in domain.RateLimitReservationInput) (domain.RateLimitReservation, error) {
	if in.RequestID == "" || in.UserID <= 0 || in.APIKeyID <= 0 || in.EstimatedTokens < 0 || !in.ExpiresAt.After(s.now()) {
		return domain.RateLimitReservation{}, store.ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextRateLimitReservationID
	s.nextRateLimitReservationID++
	s.rateLimitReservations[id] = in
	return domain.RateLimitReservation{ID: id}, nil
}
func (s *Store) FinalizeRateLimit(_ context.Context, id int64, _ int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rateLimitReservations[id]; !ok {
		return store.ErrNotFound
	}
	delete(s.rateLimitReservations, id)
	return nil
}
func (s *Store) ReleaseRateLimit(_ context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rateLimitReservations[id]; !ok {
		return store.ErrNotFound
	}
	delete(s.rateLimitReservations, id)
	return nil
}
func (s *Store) ReapRateLimitReservations(_ context.Context, limit int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	now := time.Now()
	for id, in := range s.rateLimitReservations {
		if n >= limit {
			break
		}
		if !in.ExpiresAt.After(now) {
			delete(s.rateLimitReservations, id)
			n++
		}
	}
	return n, nil
}

func (s *Store) CountActiveRateLimitReservations(_ context.Context, userID int, apiKeyID *int, model string, channelID *int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var count int64
	for _, in := range s.rateLimitReservations {
		if in.UserID != userID || (apiKeyID != nil && in.APIKeyID != *apiKeyID) || (model != "" && in.Model != model) || (channelID != nil && (in.ChannelID == nil || *in.ChannelID != *channelID)) {
			continue
		}
		count++
	}
	return count, nil
}
