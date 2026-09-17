package postgres

import (
	"context"
	"errors"
	"time"

	"LLMGateway/server/internal/db/sqlc"
	"LLMGateway/server/internal/domain"
	"LLMGateway/server/internal/store"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Store) GetChannelHealth(channelID int) (domain.ChannelHealth, error) {
	row, err := s.queries.GetChannelHealth(context.Background(), int64(channelID))
	if err != nil {
		if errors.Is(mapError(err), store.ErrNotFound) {
			return store.NewChannelHealth(channelID), nil
		}
		return domain.ChannelHealth{}, mapError(err)
	}
	return store.EvaluateChannelHealth(channelHealthFromRow(row), s.now(), s.breaker), nil
}

func (s *Store) RecordChannelSuccess(channelID int) (domain.ChannelHealth, error) {
	current, err := s.GetChannelHealth(channelID)
	if err != nil {
		return domain.ChannelHealth{}, err
	}
	next := store.ApplyChannelSuccess(current, s.now())
	if err := s.upsertChannelHealth(next); err != nil {
		return domain.ChannelHealth{}, err
	}
	return next, nil
}

func (s *Store) RecordChannelFailure(channelID int, reason string) (domain.ChannelHealth, error) {
	current, err := s.GetChannelHealth(channelID)
	if err != nil {
		return domain.ChannelHealth{}, err
	}
	next := store.ApplyChannelFailure(current, reason, s.now(), s.breaker)
	if err := s.upsertChannelHealth(next); err != nil {
		return domain.ChannelHealth{}, err
	}
	return next, nil
}

func (s *Store) ResetChannelHealth(channelID int) error {
	if _, err := s.queries.DeleteChannelHealth(context.Background(), int64(channelID)); err != nil {
		return mapError(err)
	}
	return nil
}

func (s *Store) ListChannelHealth() (domain.ListResponse, error) {
	rows, err := s.queries.ListChannelHealth(context.Background())
	if err != nil {
		return domain.ListResponse{}, mapError(err)
	}
	list := []any{}
	for _, row := range rows {
		health := store.EvaluateChannelHealth(channelHealthFromRow(row), s.now(), s.breaker)
		list = append(list, channelHealthDTO(&health))
	}
	return domain.ListResponse{List: list, Total: len(list)}, nil
}

func (s *Store) upsertChannelHealth(health domain.ChannelHealth) error {
	var openedAt pgtype.Timestamptz
	if health.OpenedAt != nil {
		if parsed, err := time.Parse(time.RFC3339, *health.OpenedAt); err == nil {
			openedAt = pgtype.Timestamptz{Time: parsed, Valid: true}
		}
	}
	return mapError(s.queries.UpsertChannelHealth(context.Background(), sqlc.UpsertChannelHealthParams{
		ChannelID:           int64(health.ChannelID),
		State:               string(health.State),
		ConsecutiveFailures: int32(health.ConsecutiveFailures),
		SuccessCount:        health.SuccessCount,
		FailureCount:        health.FailureCount,
		OpenedAt:            openedAt,
	}))
}

func channelHealthFromRow(row sqlc.ChannelHealth) domain.ChannelHealth {
	updatedAt := ""
	if row.UpdatedAt.Valid {
		updatedAt = row.UpdatedAt.Time.UTC().Format(time.RFC3339)
	}
	return domain.ChannelHealth{
		ChannelID:           int(row.ChannelID),
		State:               domain.HealthState(row.State),
		ConsecutiveFailures: int(row.ConsecutiveFailures),
		SuccessCount:        row.SuccessCount,
		FailureCount:        row.FailureCount,
		OpenedAt:            optionalTimestamp(row.OpenedAt),
		UpdatedAt:           updatedAt,
	}
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
