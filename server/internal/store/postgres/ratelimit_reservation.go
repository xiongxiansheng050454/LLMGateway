package postgres

import (
	"context"

	domain "LLMGateway/server/internal/ratelimit"
)

func (s *Store) InsertRateLimitReservation(in domain.RateLimitReservationInput) (int64, error) {
	var id int64
	err := s.pool.QueryRow(context.Background(), `INSERT INTO rate_limit_reservations(request_id,user_id,api_key_id,model,channel_id,estimated_tokens,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, in.RequestID, in.UserID, in.APIKeyID, in.Model, in.ChannelID, in.EstimatedTokens, in.ExpiresAt.UTC()).Scan(&id)
	if err != nil {
		return 0, mapError(err)
	}
	return id, nil
}

func (s *Store) FinalizeRateLimitReservation(id int64) (bool, error) {
	result, err := s.pool.Exec(context.Background(), `UPDATE rate_limit_reservations SET status='settled' WHERE id=$1 AND status='pending'`, id)
	if err != nil {
		return false, mapError(err)
	}
	return result.RowsAffected() > 0, nil
}

func (s *Store) ReleaseRateLimitReservation(id int64) (bool, error) {
	result, err := s.pool.Exec(context.Background(), `UPDATE rate_limit_reservations SET status='released',released_at=now() WHERE id=$1 AND status='pending'`, id)
	if err != nil {
		return false, mapError(err)
	}
	return result.RowsAffected() > 0, nil
}

func (s *Store) ReapRateLimitReservations(limit int) (int, error) {
	result, err := s.pool.Exec(context.Background(), `WITH expired AS (SELECT id FROM rate_limit_reservations WHERE status='pending' AND expires_at<=now() ORDER BY id LIMIT $1) UPDATE rate_limit_reservations SET status='expired' WHERE id IN (SELECT id FROM expired)`, limit)
	if err != nil {
		return 0, mapError(err)
	}
	return int(result.RowsAffected()), nil
}

func (s *Store) CountActiveRateLimitReservations(userID int, apiKeyID *int, model string, channelID *int) (int64, error) {
	var count int64
	err := s.pool.QueryRow(context.Background(), `SELECT count(*) FROM rate_limit_reservations WHERE status='pending' AND user_id=$1 AND ($2=0 OR api_key_id=$2) AND ($3='' OR model=$3) AND ($4=0 OR channel_id=$4)`, userID, optionalID(apiKeyID), model, optionalID(channelID)).Scan(&count)
	if err != nil {
		return 0, mapError(err)
	}
	return count, nil
}
