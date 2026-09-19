package postgres

import (
	"context"
	"fmt"
	"time"

	"LLMGateway/server/internal/db/sqlc"
	"LLMGateway/server/internal/money"
	domain "LLMGateway/server/internal/quota"
	"LLMGateway/server/internal/store"
	usage "LLMGateway/server/internal/usage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type quotaPolicyRow struct {
	id         int64
	name       string
	scopeType  domain.QuotaScopeType
	scopeID    int
	periodType domain.QuotaPeriodType
	tokenLimit *int64
	costLimit  *string
	enabled    bool
}

type quotaItemRow struct {
	policyID      int64
	periodStart   time.Time
	reservedToken int64
	reservedCost  string
}

func (s *Store) CreateQuotaPolicy(in domain.QuotaPolicyInput) (domain.QuotaPolicyDTO, error) {
	policy, err := domain.NormalizeQuotaPolicy(in, nil)
	if err != nil {
		return domain.QuotaPolicyDTO{}, err
	}
	params := sqlc.CreateQuotaPolicyParams{PolicyName: policy.PolicyName, ScopeType: string(policy.ScopeType), PeriodType: string(policy.PeriodType), Enabled: policy.Enabled}
	if policy.ScopeType == domain.QuotaScopeUser {
		params.UserID = pgtype.Int8{Int64: int64(policy.ScopeID), Valid: true}
	} else {
		params.ApiKeyID = pgtype.Int8{Int64: int64(policy.ScopeID), Valid: true}
	}
	if policy.TokenLimit != nil {
		params.TokenLimit = pgtype.Int8{Int64: *policy.TokenLimit, Valid: true}
	}
	params.CostLimit = numericValue(policy.CostLimit)
	id, err := s.queries.CreateQuotaPolicy(context.Background(), params)
	if err != nil {
		return domain.QuotaPolicyDTO{}, mapError(err)
	}
	return s.getQuotaPolicy(int(id))
}

func (s *Store) UpdateQuotaPolicy(id int, in domain.QuotaPolicyInput) (domain.QuotaPolicyDTO, error) {
	existingDTO, err := s.getQuotaPolicy(id)
	if err != nil {
		return domain.QuotaPolicyDTO{}, err
	}
	existing := domain.QuotaPolicy{ID: existingDTO.ID, PolicyName: existingDTO.PolicyName, ScopeType: existingDTO.ScopeType, ScopeID: existingDTO.ScopeID, PeriodType: existingDTO.PeriodType, TokenLimit: existingDTO.TokenLimit, CostLimit: existingDTO.CostLimit, Enabled: existingDTO.Enabled}
	policy, err := domain.NormalizeQuotaPolicy(in, &existing)
	if err != nil {
		return domain.QuotaPolicyDTO{}, err
	}
	if in.ScopeType != nil || in.ScopeID != nil || in.PeriodType != nil {
		if policy.ScopeType != existing.ScopeType || policy.ScopeID != existing.ScopeID || policy.PeriodType != existing.PeriodType {
			return domain.QuotaPolicyDTO{}, fmt.Errorf("%w: quota scope and period cannot be changed", store.ErrInvalid)
		}
	}
	params := sqlc.UpdateQuotaPolicyParams{ID: int64(id), PolicyName: policy.PolicyName, Enabled: policy.Enabled, CostLimit: numericValue(policy.CostLimit)}
	if policy.TokenLimit != nil {
		params.TokenLimit = pgtype.Int8{Int64: *policy.TokenLimit, Valid: true}
	}
	affected, err := s.queries.UpdateQuotaPolicy(context.Background(), params)
	if err != nil {
		return domain.QuotaPolicyDTO{}, mapError(err)
	}
	if affected == 0 {
		return domain.QuotaPolicyDTO{}, store.ErrNotFound
	}
	return s.getQuotaPolicy(id)
}

func (s *Store) DeleteQuotaPolicy(id int) error {
	affected, err := s.queries.DeleteQuotaPolicy(context.Background(), int64(id))
	if err != nil {
		return mapError(err)
	}
	if affected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *Store) getQuotaPolicy(id int) (domain.QuotaPolicyDTO, error) {
	row, err := s.queries.GetQuotaPolicy(context.Background(), int64(id))
	if err != nil {
		return domain.QuotaPolicyDTO{}, mapError(err)
	}
	scopeID := optionalInt(row.UserID)
	if row.ScopeType == string(domain.QuotaScopeAPIKey) {
		scopeID = optionalInt(row.ApiKeyID)
	}
	if scopeID == nil {
		return domain.QuotaPolicyDTO{}, fmt.Errorf("%w: quota scope missing", store.ErrInvalid)
	}
	var tokenLimit *int64
	if row.TokenLimit.Valid {
		value := row.TokenLimit.Int64
		tokenLimit = &value
	}
	return domain.QuotaPolicyDTO{ID: int(row.ID), PolicyName: row.PolicyName, ScopeType: domain.QuotaScopeType(row.ScopeType), ScopeID: *scopeID, PeriodType: domain.QuotaPeriodType(row.PeriodType), TokenLimit: tokenLimit, CostLimit: optionalString(row.CostLimit), Enabled: row.Enabled}, nil
}

func (s *Store) ListQuotaPolicies(filter domain.QuotaPolicyFilter) (domain.ListResponse[domain.QuotaPolicyDTO], error) {
	ctx := context.Background()
	limit, offset := limitOffset(filter.Page, filter.PageSize)
	rows, err := s.pool.Query(ctx, `
SELECT id, policy_name, scope_type, COALESCE(user_id, api_key_id), period_type,
       token_limit, cost_limit::text, enabled
FROM quota_policies
WHERE deleted_at IS NULL
  AND ($1 = '' OR scope_type = $1)
  AND ($2 = 0 OR COALESCE(user_id, api_key_id) = $2)
  AND ($3::boolean IS NULL OR enabled = $3)
ORDER BY id
LIMIT $4 OFFSET $5`, filter.ScopeType, filter.ScopeID, filter.Enabled, limit, offset)
	if err != nil {
		return domain.ListResponse[domain.QuotaPolicyDTO]{}, mapError(err)
	}
	defer rows.Close()
	list := []domain.QuotaPolicyDTO{}
	for rows.Next() {
		var row quotaPolicyRow
		var scopeType, periodType string
		var token pgtype.Int8
		var cost pgtype.Text
		if err := rows.Scan(&row.id, &row.name, &scopeType, &row.scopeID, &periodType, &token, &cost, &row.enabled); err != nil {
			return domain.ListResponse[domain.QuotaPolicyDTO]{}, mapError(err)
		}
		row.scopeType, row.periodType = domain.QuotaScopeType(scopeType), domain.QuotaPeriodType(periodType)
		if token.Valid {
			value := token.Int64
			row.tokenLimit = &value
		}
		row.costLimit = optionalString(textOrEmpty(cost))
		list = append(list, domain.QuotaPolicyToDTO(domain.QuotaPolicy{ID: int(row.id), PolicyName: row.name, ScopeType: row.scopeType, ScopeID: row.scopeID, PeriodType: row.periodType, TokenLimit: row.tokenLimit, CostLimit: row.costLimit, Enabled: row.enabled}))
	}
	if err := rows.Err(); err != nil {
		return domain.ListResponse[domain.QuotaPolicyDTO]{}, mapError(err)
	}
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM quota_policies WHERE deleted_at IS NULL AND ($1 = '' OR scope_type = $1) AND ($2 = 0 OR COALESCE(user_id, api_key_id) = $2) AND ($3::boolean IS NULL OR enabled = $3)`, filter.ScopeType, filter.ScopeID, filter.Enabled).Scan(&total); err != nil {
		return domain.ListResponse[domain.QuotaPolicyDTO]{}, mapError(err)
	}
	return domain.ListResponse[domain.QuotaPolicyDTO]{List: list, Total: total}, nil
}

func (s *Store) ListQuotaUsage(ctx context.Context, filter domain.QuotaPolicyFilter) (domain.ListResponse[domain.QuotaUsageDTO], error) {
	limit, offset := limitOffset(filter.Page, filter.PageSize)
	rows, err := s.pool.Query(ctx, `
SELECT p.id, p.policy_name, p.scope_type, COALESCE(p.user_id, p.api_key_id), p.period_type,
       b.period_start, b.period_end, p.token_limit, b.used_tokens, b.reserved_tokens,
       p.cost_limit::text, b.used_cost::text, b.reserved_cost::text
FROM quota_policies p JOIN quota_buckets b ON b.policy_id = p.id
WHERE p.deleted_at IS NULL AND ($1 = '' OR p.scope_type = $1) AND ($2 = 0 OR COALESCE(p.user_id, p.api_key_id) = $2)
  AND b.period_start <= now() AND b.period_end > now()
ORDER BY p.id
LIMIT $3 OFFSET $4`, filter.ScopeType, filter.ScopeID, limit, offset)
	if err != nil {
		return domain.ListResponse[domain.QuotaUsageDTO]{}, mapError(err)
	}
	defer rows.Close()
	list := []domain.QuotaUsageDTO{}
	for rows.Next() {
		var item domain.QuotaUsageDTO
		var scopeType, periodType string
		var start, end time.Time
		var token pgtype.Int8
		var cost pgtype.Text
		if err := rows.Scan(&item.PolicyID, &item.PolicyName, &scopeType, &item.ScopeID, &periodType, &start, &end, &token, &item.UsedTokens, &item.ReservedTokens, &cost, &item.UsedCost, &item.ReservedCost); err != nil {
			return domain.ListResponse[domain.QuotaUsageDTO]{}, mapError(err)
		}
		item.ScopeType, item.PeriodType = domain.QuotaScopeType(scopeType), domain.QuotaPeriodType(periodType)
		item.PeriodStart, item.PeriodEnd = start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339)
		if token.Valid {
			value := token.Int64
			item.TokenLimit = &value
		}
		item.CostLimit = optionalString(textOrEmpty(cost))
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return domain.ListResponse[domain.QuotaUsageDTO]{}, mapError(err)
	}
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM quota_policies p JOIN quota_buckets b ON b.policy_id=p.id WHERE p.deleted_at IS NULL AND ($1='' OR p.scope_type=$1) AND ($2=0 OR COALESCE(p.user_id,p.api_key_id)=$2) AND b.period_start <= now() AND b.period_end > now()`, filter.ScopeType, filter.ScopeID).Scan(&total); err != nil {
		return domain.ListResponse[domain.QuotaUsageDTO]{}, mapError(err)
	}
	return domain.ListResponse[domain.QuotaUsageDTO]{List: list, Total: total}, nil
}

func (s *Store) ReserveQuota(ctx context.Context, in domain.QuotaReserveInput) (domain.QuotaReservation, error) {
	if in.RequestID == "" || in.UserID <= 0 || in.APIKeyID <= 0 || in.EstimatedTokens < 0 || !in.ExpiresAt.After(s.now()) {
		return domain.QuotaReservation{}, fmt.Errorf("%w: invalid quota reservation", store.ErrInvalid)
	}
	cost, err := money.Parse6(in.EstimatedCost)
	if err != nil || cost.Cmp(0) < 0 {
		return domain.QuotaReservation{}, fmt.Errorf("%w: invalid estimated cost", store.ErrInvalid)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.QuotaReservation{}, mapError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	policies, err := applicableQuotaPolicies(ctx, tx, in.UserID, in.APIKeyID)
	if err != nil {
		return domain.QuotaReservation{}, err
	}
	// Policy locks precede reservation/bucket locks everywhere identity deletion
	// and admission can overlap, preventing reverse lock-order deadlocks.
	if _, err := reapExpiredQuotaTx(ctx, tx, s.now(), 16, in.UserID, in.APIKeyID); err != nil {
		return domain.QuotaReservation{}, err
	}
	if len(policies) == 0 {
		return domain.QuotaReservation{}, tx.Commit(ctx)
	}
	var reservationID int64
	err = tx.QueryRow(ctx, `INSERT INTO quota_reservations (request_id, user_id, api_key_id, model, estimated_tokens, estimated_cost, expires_at) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, in.RequestID, in.UserID, in.APIKeyID, in.Model, in.EstimatedTokens, money.Format6(cost), in.ExpiresAt.UTC()).Scan(&reservationID)
	if err != nil {
		return domain.QuotaReservation{}, mapError(err)
	}
	now := s.now().UTC()
	for _, policy := range policies {
		start, end, err := domain.QuotaPeriodBounds(now, policy.periodType)
		if err != nil {
			return domain.QuotaReservation{}, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO quota_buckets (policy_id, period_start, period_end) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, policy.id, start, end); err != nil {
			return domain.QuotaReservation{}, mapError(err)
		}
		var lockedPolicyID int64
		if err := tx.QueryRow(ctx, `SELECT policy_id FROM quota_buckets WHERE policy_id=$1 AND period_start=$2 FOR UPDATE`, policy.id, start).Scan(&lockedPolicyID); err != nil {
			return domain.QuotaReservation{}, mapError(err)
		}
		result, err := tx.Exec(ctx, `
UPDATE quota_buckets b
SET reserved_tokens = b.reserved_tokens + $3,
    reserved_cost = b.reserved_cost + $4,
    updated_at = now()
FROM quota_policies p
WHERE b.policy_id = $1 AND b.period_start = $2 AND p.id = b.policy_id
  AND p.enabled = true AND p.deleted_at IS NULL
  AND (p.token_limit IS NULL OR b.used_tokens + b.reserved_tokens + $3 <= p.token_limit)
  AND (p.cost_limit IS NULL OR b.used_cost + b.reserved_cost + $4 <= p.cost_limit)`, policy.id, start, in.EstimatedTokens, money.Format6(cost))
		if err != nil {
			return domain.QuotaReservation{}, mapError(err)
		}
		if result.RowsAffected() != 1 {
			return domain.QuotaReservation{}, store.ErrQuotaExceeded
		}
		if _, err := tx.Exec(ctx, `INSERT INTO quota_reservation_items (reservation_id, policy_id, period_start, reserved_tokens, reserved_cost) VALUES ($1,$2,$3,$4,$5)`, reservationID, policy.id, start, in.EstimatedTokens, money.Format6(cost)); err != nil {
			return domain.QuotaReservation{}, mapError(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.QuotaReservation{}, mapError(err)
	}
	return domain.QuotaReservation{ID: reservationID, RequestID: in.RequestID, EstimatedTokens: in.EstimatedTokens, EstimatedCost: money.Format6(cost)}, nil
}

func (s *Store) ReleaseQuota(ctx context.Context, reservationID int64) error {
	if reservationID == 0 {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return mapError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := releaseQuotaTx(ctx, tx, reservationID, "released", s.now()); err != nil {
		return err
	}
	return mapError(tx.Commit(ctx))
}

func (s *Store) ReapExpiredQuotaReservations(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 100
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, mapError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	count, err := reapExpiredQuotaTx(ctx, tx, s.now(), limit, 0, 0)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, mapError(err)
	}
	return count, nil
}

func applicableQuotaPolicies(ctx context.Context, tx pgx.Tx, userID, keyID int) ([]quotaPolicyRow, error) {
	rows, err := tx.Query(ctx, `SELECT id, policy_name, scope_type, COALESCE(user_id, api_key_id), period_type, token_limit, cost_limit::text, enabled FROM quota_policies WHERE deleted_at IS NULL AND enabled=true AND ((scope_type='user' AND user_id=$1) OR (scope_type='api_key' AND api_key_id=$2)) ORDER BY id FOR SHARE`, userID, keyID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	var result []quotaPolicyRow
	for rows.Next() {
		var row quotaPolicyRow
		var scopeType, periodType string
		var token pgtype.Int8
		var cost pgtype.Text
		if err := rows.Scan(&row.id, &row.name, &scopeType, &row.scopeID, &periodType, &token, &cost, &row.enabled); err != nil {
			return nil, mapError(err)
		}
		row.scopeType, row.periodType = domain.QuotaScopeType(scopeType), domain.QuotaPeriodType(periodType)
		if token.Valid {
			value := token.Int64
			row.tokenLimit = &value
		}
		row.costLimit = optionalString(textOrEmpty(cost))
		result = append(result, row)
	}
	return result, mapError(rows.Err())
}

func releaseQuotaTx(ctx context.Context, tx pgx.Tx, reservationID int64, status string, now time.Time) error {
	var current string
	if err := tx.QueryRow(ctx, `SELECT status FROM quota_reservations WHERE id=$1 FOR UPDATE`, reservationID).Scan(&current); err != nil {
		return mapError(err)
	}
	if current != "pending" {
		return nil
	}
	items, err := quotaReservationItems(ctx, tx, reservationID)
	if err != nil {
		return err
	}
	for _, item := range items {
		result, err := tx.Exec(ctx, `UPDATE quota_buckets SET reserved_tokens=reserved_tokens-$3, reserved_cost=reserved_cost-$4, updated_at=now() WHERE policy_id=$1 AND period_start=$2 AND reserved_tokens >= $3 AND reserved_cost >= $4`, item.policyID, item.periodStart, item.reservedToken, item.reservedCost)
		if err != nil {
			return mapError(err)
		}
		if result.RowsAffected() != 1 {
			return fmt.Errorf("%w: invalid quota bucket reservation", store.ErrInvalid)
		}
	}
	_, err = tx.Exec(ctx, `UPDATE quota_reservations SET status=$2, released_at=$3, updated_at=now() WHERE id=$1`, reservationID, status, now.UTC())
	return mapError(err)
}

func reapExpiredQuotaTx(ctx context.Context, tx pgx.Tx, now time.Time, limit, userID, keyID int) (int, error) {
	rows, err := tx.Query(ctx, `SELECT id FROM quota_reservations WHERE status='pending' AND expires_at <= $1 AND ($2=0 OR user_id=$2 OR api_key_id=$3) ORDER BY expires_at,id FOR UPDATE SKIP LOCKED LIMIT $4`, now.UTC(), userID, keyID, limit)
	if err != nil {
		return 0, mapError(err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, mapError(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if err := releaseQuotaTx(ctx, tx, id, "expired", now); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

func quotaReservationItems(ctx context.Context, tx pgx.Tx, reservationID int64) ([]quotaItemRow, error) {
	rows, err := tx.Query(ctx, `SELECT policy_id, period_start, reserved_tokens, reserved_cost::text FROM quota_reservation_items WHERE reservation_id=$1 ORDER BY policy_id FOR UPDATE`, reservationID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	var items []quotaItemRow
	for rows.Next() {
		var item quotaItemRow
		if err := rows.Scan(&item.policyID, &item.periodStart, &item.reservedToken, &item.reservedCost); err != nil {
			return nil, mapError(err)
		}
		items = append(items, item)
	}
	return items, mapError(rows.Err())
}

func settleQuotaTx(ctx context.Context, tx pgx.Tx, in usage.ChatSettlementInput, actualTokens int64, actualCost string, now time.Time) error {
	reservationID := in.ReservationID
	if reservationID == 0 {
		return nil
	}
	var status, requestID string
	var userID, keyID int
	if err := tx.QueryRow(ctx, `SELECT status, request_id, user_id, api_key_id FROM quota_reservations WHERE id=$1 FOR UPDATE`, reservationID).Scan(&status, &requestID, &userID, &keyID); err != nil {
		return mapError(err)
	}
	if status == "settled" {
		return fmt.Errorf("%w: quota reservation already settled", store.ErrInvalid)
	}
	if status != "pending" {
		return fmt.Errorf("%w: quota reservation is %s", store.ErrInvalid, status)
	}
	if requestID != in.UsageLog.RequestID || userID != in.UserID || keyID != in.APIKeyID {
		return fmt.Errorf("%w: quota reservation identity mismatch", store.ErrInvalid)
	}
	items, err := quotaReservationItems(ctx, tx, reservationID)
	if err != nil {
		return err
	}
	for _, item := range items {
		result, err := tx.Exec(ctx, `UPDATE quota_buckets SET reserved_tokens=reserved_tokens-$3, reserved_cost=reserved_cost-$4, used_tokens=used_tokens+$5, used_cost=used_cost+$6, updated_at=now() WHERE policy_id=$1 AND period_start=$2 AND reserved_tokens >= $3 AND reserved_cost >= $4 AND $5 <= $3 AND $6 <= $4`, item.policyID, item.periodStart, item.reservedToken, item.reservedCost, actualTokens, actualCost)
		if err != nil {
			return mapError(err)
		}
		if result.RowsAffected() != 1 {
			return fmt.Errorf("%w: invalid quota bucket settlement", store.ErrInvalid)
		}
		if _, err := tx.Exec(ctx, `UPDATE quota_reservation_items SET actual_tokens=$3, actual_cost=$4 WHERE reservation_id=$1 AND policy_id=$2`, reservationID, item.policyID, actualTokens, actualCost); err != nil {
			return mapError(err)
		}
	}
	_, err = tx.Exec(ctx, `UPDATE quota_reservations SET status='settled', actual_tokens=$2, actual_cost=$3, settled_at=$4, updated_at=now() WHERE id=$1`, reservationID, actualTokens, actualCost, now.UTC())
	return mapError(err)
}

func cleanupQuotaReservationsTx(ctx context.Context, tx pgx.Tx, userID, keyID int) error {
	if _, err := tx.Exec(ctx, `SELECT id FROM quota_policies WHERE deleted_at IS NULL AND (($1 > 0 AND scope_type='user' AND user_id=$1) OR ($2 > 0 AND scope_type='api_key' AND api_key_id=$2) OR ($1 > 0 AND scope_type='api_key' AND api_key_id IN (SELECT id FROM client_api_keys WHERE user_id=$1))) ORDER BY id FOR UPDATE`, userID, keyID); err != nil {
		return mapError(err)
	}
	rows, err := tx.Query(ctx, `SELECT id FROM quota_reservations WHERE ($1=0 OR user_id=$1) AND ($2=0 OR api_key_id=$2) ORDER BY id FOR UPDATE`, userID, keyID)
	if err != nil {
		return mapError(err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return mapError(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if err := releaseQuotaTx(ctx, tx, id, "released", time.Now()); err != nil {
			return err
		}
	}
	if keyID > 0 {
		_, err = tx.Exec(ctx, `DELETE FROM quota_reservations WHERE api_key_id=$1 AND ($2=0 OR user_id=$2)`, keyID, userID)
	} else {
		_, err = tx.Exec(ctx, `DELETE FROM quota_reservations WHERE user_id=$1`, userID)
	}
	return mapError(err)
}

func numericValue(value *string) pgtype.Numeric {
	if value == nil || *value == "" {
		return pgtype.Numeric{}
	}
	var numeric pgtype.Numeric
	_ = numeric.Scan(*value)
	return numeric
}
