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
	if !reason.CountsAsChannelFailure() {
		return s.GetChannelHealth(channelID)
	}
	return s.recordChannelHealth(channelID, func(current domain.ChannelHealth) domain.ChannelHealth {
		return domain.ApplyChannelFailure(current, reason, s.now(), s.breaker)
	})
}

func (s *Store) RecordChannelAttempt(_ context.Context, channelID int, success bool, reason domain.FailureReason) (domain.ChannelHealth, error) {
	if success {
		return s.RecordChannelSuccess(channelID)
	}
	return s.RecordChannelFailure(channelID, reason)
}

func (s *Store) AcquireChannelProbe(ctx context.Context, channelID int, lease time.Duration) (bool, error) {
	result, err := s.pool.Exec(ctx, `INSERT INTO channel_breaker_probes(channel_id, lease_id, leased_until) VALUES($1, gen_random_uuid(), now()+$2::int*interval '1 second') ON CONFLICT(channel_id) DO UPDATE SET lease_id=gen_random_uuid(), leased_until=EXCLUDED.leased_until WHERE channel_breaker_probes.leased_until <= now()`, channelID, int(lease.Seconds()))
	if err != nil {
		return false, mapError(err)
	}
	return result.RowsAffected() == 1, nil
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
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return mapError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, table := range []string{"channel_breaker_probes", "channel_breaker_configs", "channel_health_buckets", "channel_health"} {
		if _, err := tx.Exec(ctx, "DELETE FROM "+table+" WHERE channel_id=$1", channelID); err != nil {
			return mapError(err)
		}
	}
	return mapError(tx.Commit(ctx))
}

func (s *Store) ListChannelHealth() (domain.ListResponse[domain.ChannelHealthDTO], error) {
	rows, err := s.pool.Query(context.Background(), `SELECT c.id, COALESCE(h.state,'closed'), COALESCE(h.consecutive_failures,0), COALESCE(h.success_count,0), COALESCE(h.failure_count,0), h.opened_at, COALESCE(h.updated_at,c.updated_at) FROM channels c LEFT JOIN channel_health h ON h.channel_id=c.id ORDER BY c.id`)
	if err != nil {
		return domain.ListResponse[domain.ChannelHealthDTO]{}, mapError(err)
	}
	defer rows.Close()
	list := []domain.ChannelHealthDTO{}
	for rows.Next() {
		var h domain.ChannelHealth
		var opened, updated pgtype.Timestamptz
		if err := rows.Scan(&h.ChannelID, &h.State, &h.ConsecutiveFailures, &h.SuccessCount, &h.FailureCount, &opened, &updated); err != nil {
			return domain.ListResponse[domain.ChannelHealthDTO]{}, mapError(err)
		}
		h.OpenedAt = optionalTimestamp(opened)
		h.UpdatedAt = updated.Time.UTC().Format(time.RFC3339)
		h = domain.EvaluateChannelHealth(h, s.now(), s.breaker)
		list = append(list, channelHealthDTO(&h))
	}
	if err := rows.Err(); err != nil {
		return domain.ListResponse[domain.ChannelHealthDTO]{}, mapError(err)
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
