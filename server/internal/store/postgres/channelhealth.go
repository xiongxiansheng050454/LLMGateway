package postgres

import (
	"context"
	"errors"
	"time"

	domain "LLMGateway/server/internal/catalog"
	"LLMGateway/server/internal/db/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Store) GetChannelHealthRow(channelID int) (domain.ChannelHealth, bool, error) {
	row, err := s.queries.GetChannelHealth(context.Background(), int64(channelID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ChannelHealth{}, false, nil
		}
		return domain.ChannelHealth{}, false, mapError(err)
	}
	return channelHealthFromRow(row), true, nil
}

func (s *Store) ListChannelHealthRows() ([]domain.ChannelHealth, error) {
	rows, err := s.pool.Query(context.Background(), `SELECT c.id, COALESCE(h.state,'closed'), COALESCE(h.consecutive_failures,0), COALESCE(h.success_count,0), COALESCE(h.failure_count,0), h.opened_at, COALESCE(h.updated_at,c.updated_at) FROM channels c LEFT JOIN channel_health h ON h.channel_id=c.id ORDER BY c.id`)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	list := []domain.ChannelHealth{}
	for rows.Next() {
		var h domain.ChannelHealth
		var opened, updated pgtype.Timestamptz
		if err := rows.Scan(&h.ChannelID, &h.State, &h.ConsecutiveFailures, &h.SuccessCount, &h.FailureCount, &opened, &updated); err != nil {
			return nil, mapError(err)
		}
		h.OpenedAt = optionalTimestamp(opened)
		h.UpdatedAt = updated.Time.UTC().Format(time.RFC3339)
		list = append(list, h)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}
	return list, nil
}

func (s *Store) AcquireChannelProbe(ctx context.Context, channelID int, lease time.Duration) (bool, error) {
	result, err := s.pool.Exec(ctx, `INSERT INTO channel_breaker_probes(channel_id, lease_id, leased_until) VALUES($1, gen_random_uuid(), now()+$2::int*interval '1 second') ON CONFLICT(channel_id) DO UPDATE SET lease_id=gen_random_uuid(), leased_until=EXCLUDED.leased_until WHERE channel_breaker_probes.leased_until <= now()`, channelID, int(lease.Seconds()))
	if err != nil {
		return false, mapError(err)
	}
	return result.RowsAffected() == 1, nil
}

func (t *Tx) EnsureChannelHealth(channelID int) error {
	return mapError(t.queries.EnsureChannelHealth(context.Background(), int64(channelID)))
}

func (t *Tx) GetChannelHealthForUpdate(channelID int) (domain.ChannelHealth, error) {
	row, err := t.queries.GetChannelHealthForUpdate(context.Background(), int64(channelID))
	if err != nil {
		return domain.ChannelHealth{}, mapError(err)
	}
	return channelHealthFromRow(row), nil
}

func (t *Tx) UpdateChannelHealth(health domain.ChannelHealth) (bool, error) {
	var openedAt pgtype.Timestamptz
	if health.OpenedAt != nil {
		if parsed, err := time.Parse(time.RFC3339, *health.OpenedAt); err == nil {
			openedAt = pgtype.Timestamptz{Time: parsed, Valid: true}
		}
	}
	affected, err := t.queries.UpdateChannelHealth(context.Background(), sqlc.UpdateChannelHealthParams{
		State:               string(health.State),
		ConsecutiveFailures: int32(health.ConsecutiveFailures),
		SuccessCount:        health.SuccessCount,
		FailureCount:        health.FailureCount,
		OpenedAt:            openedAt,
		ChannelID:           int64(health.ChannelID),
	})
	if err != nil {
		return false, mapError(err)
	}
	return affected > 0, nil
}

func (t *Tx) DeleteChannelHealth(channelID int) error {
	ctx := context.Background()
	for _, table := range []string{"channel_breaker_probes", "channel_breaker_configs", "channel_health_buckets", "channel_health"} {
		if _, err := t.tx.Exec(ctx, "DELETE FROM "+table+" WHERE channel_id=$1", channelID); err != nil {
			return mapError(err)
		}
	}
	return nil
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
