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
			return domain.NewChannelHealth(channelID), nil
		}
		return domain.ChannelHealth{}, mapError(err)
	}
	return domain.EvaluateChannelHealth(channelHealthFromRow(row), s.now(), s.breaker), nil
}

func (s *Store) RecordChannelSuccess(channelID int) (domain.ChannelHealth, error) {
	return s.recordChannelHealth(channelID, func(current domain.ChannelHealth) domain.ChannelHealth {
		return domain.ApplyChannelSuccess(current, s.now())
	})
}

func (s *Store) RecordChannelFailure(channelID int, reason domain.FailureReason) (domain.ChannelHealth, error) {
	return s.recordChannelHealth(channelID, func(current domain.ChannelHealth) domain.ChannelHealth {
		return domain.ApplyChannelFailure(current, reason, s.now(), s.breaker)
	})
}

// recordChannelHealth applies a state transition inside a transaction, holding
// a row lock so concurrent recordings cannot lose updates.
func (s *Store) recordChannelHealth(channelID int, apply func(domain.ChannelHealth) domain.ChannelHealth) (domain.ChannelHealth, error) {
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ChannelHealth{}, mapError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)

	if err := queries.EnsureChannelHealth(ctx, int64(channelID)); err != nil {
		return domain.ChannelHealth{}, mapError(err)
	}
	row, err := queries.GetChannelHealthForUpdate(ctx, int64(channelID))
	if err != nil {
		return domain.ChannelHealth{}, mapError(err)
	}

	current := domain.EvaluateChannelHealth(channelHealthFromRow(row), s.now(), s.breaker)
	next := apply(current)

	var openedAt pgtype.Timestamptz
	if next.OpenedAt != nil {
		if parsed, err := time.Parse(time.RFC3339, *next.OpenedAt); err == nil {
			openedAt = pgtype.Timestamptz{Time: parsed, Valid: true}
		}
	}
	affected, err := queries.UpdateChannelHealth(ctx, sqlc.UpdateChannelHealthParams{
		State:               string(next.State),
		ConsecutiveFailures: int32(next.ConsecutiveFailures),
		SuccessCount:        next.SuccessCount,
		FailureCount:        next.FailureCount,
		OpenedAt:            openedAt,
		ChannelID:           int64(channelID),
	})
	if err != nil {
		return domain.ChannelHealth{}, mapError(err)
	}
	if affected == 0 {
		return domain.ChannelHealth{}, store.ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.ChannelHealth{}, mapError(err)
	}
	return next, nil
}

func (s *Store) ResetChannelHealth(channelID int) error {
	if _, err := s.queries.DeleteChannelHealth(context.Background(), int64(channelID)); err != nil {
		return mapError(err)
	}
	return nil
}

func (s *Store) ListChannelHealth() (domain.ListResponse[domain.ChannelHealthDTO], error) {
	rows, err := s.queries.ListChannelHealth(context.Background())
	if err != nil {
		return domain.ListResponse[domain.ChannelHealthDTO]{}, mapError(err)
	}
	list := []domain.ChannelHealthDTO{}
	for _, row := range rows {
		health := domain.EvaluateChannelHealth(channelHealthFromRow(row), s.now(), s.breaker)
		list = append(list, channelHealthDTO(&health))
	}
	return domain.ListResponse[domain.ChannelHealthDTO]{List: list, Total: len(list)}, nil
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

func channelHealthDTO(health *domain.ChannelHealth) domain.ChannelHealthDTO {
	return domain.ChannelHealthDTO{ChannelID: health.ChannelID, State: string(health.State), ConsecutiveFailures: health.ConsecutiveFailures, SuccessCount: health.SuccessCount, FailureCount: health.FailureCount, OpenedAt: health.OpenedAt, UpdatedAt: health.UpdatedAt}
}
