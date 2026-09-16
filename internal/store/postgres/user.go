package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"LLMGateway/internal/crypto"
	"LLMGateway/internal/db/sqlc"
	"LLMGateway/internal/domain"
	"LLMGateway/internal/money"
	"LLMGateway/internal/store"

	"github.com/jackc/pgx/v5/pgtype"
)

const defaultPermissions = `{"models":["*"]}`

func (s *Store) ListUsers(page, pageSize int) (domain.ListResponse, error) {
	ctx := context.Background()
	limit, offset := limitOffset(page, pageSize)

	rows, err := s.queries.ListUsers(ctx, sqlc.ListUsersParams{Limit: limit, Offset: offset})
	if err != nil {
		return domain.ListResponse{}, mapError(err)
	}
	total, err := s.queries.CountUsers(ctx)
	if err != nil {
		return domain.ListResponse{}, mapError(err)
	}

	list := []any{}
	for _, row := range rows {
		list = append(list, userDTO(row.ID, row.Nickname, row.UserGroup, row.Status, textValue(row.AvailableBalance), textValue(row.FrozenBalance)))
	}
	return domain.ListResponse{List: list, Total: int(total)}, nil
}

func (s *Store) CreateUser(in domain.UserInput) (map[string]any, error) {
	if in.UserGroup == "" {
		in.UserGroup = "default"
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if in.Status != "active" && in.Status != "suspended" {
		return nil, fmt.Errorf("%w: invalid status", store.ErrInvalid)
	}

	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)

	id, err := queries.CreateUser(ctx, sqlc.CreateUserParams{Nickname: in.Nickname, UserGroup: in.UserGroup, Status: in.Status})
	if err != nil {
		return nil, mapError(err)
	}
	if err := queries.CreateUserBalance(ctx, id); err != nil {
		return nil, mapError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, mapError(err)
	}
	return s.getUserDTO(id)
}

func (s *Store) UpdateUser(id int, in domain.UserInput) (map[string]any, error) {
	ctx := context.Background()
	current, err := s.queries.GetUser(ctx, int64(id))
	if err != nil {
		return nil, mapError(err)
	}
	group := in.UserGroup
	if group == "" {
		group = current.UserGroup
	}

	affected, err := s.queries.UpdateUser(ctx, sqlc.UpdateUserParams{Nickname: in.Nickname, UserGroup: group, ID: int64(id)})
	if err != nil {
		return nil, mapError(err)
	}
	if affected == 0 {
		return nil, store.ErrNotFound
	}
	return s.getUserDTO(int64(id))
}

func (s *Store) UpdateUserStatus(id int, status string) (map[string]any, error) {
	if status != "active" && status != "suspended" {
		return nil, fmt.Errorf("%w: invalid status", store.ErrInvalid)
	}
	affected, err := s.queries.UpdateUserStatus(context.Background(), sqlc.UpdateUserStatusParams{Status: status, ID: int64(id)})
	if err != nil {
		return nil, mapError(err)
	}
	if affected == 0 {
		return nil, store.ErrNotFound
	}
	return s.getUserDTO(int64(id))
}

func (s *Store) DeleteUser(id int) error {
	affected, err := s.queries.DeleteUser(context.Background(), int64(id))
	if err != nil {
		return mapError(err)
	}
	if affected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *Store) RechargeUser(id int, in domain.RechargeInput) (map[string]any, error) {
	amount, err := money.Parse6(in.Amount)
	if err != nil || amount.Cmp(0) <= 0 {
		return nil, fmt.Errorf("%w: invalid amount", store.ErrInvalid)
	}

	ctx := context.Background()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)

	if _, err := queries.LockUserBalance(ctx, int64(id)); err != nil {
		return nil, mapError(err)
	}

	// Idempotency is scoped per user and evaluated after the balance row lock,
	// so concurrent repeats of the same order serialize and return the original
	// result instead of a unique-violation error.
	if in.RelatedOrderID != "" {
		existing, err := queries.GetBalanceTransactionByOrder(ctx, sqlc.GetBalanceTransactionByOrderParams{UserID: int64(id), RelatedOrderID: pgtype.Text{String: in.RelatedOrderID, Valid: true}})
		if err == nil {
			return map[string]any{"balance_after": existing.BalanceAfter}, nil
		}
		if !errors.Is(mapError(err), store.ErrNotFound) {
			return nil, mapError(err)
		}
	}
	balanceRow, err := queries.GetUserBalanceText(ctx, int64(id))
	if err != nil {
		return nil, mapError(err)
	}
	current, err := money.Parse6(textValue(balanceRow.AvailableBalance))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid balance", store.ErrInvalid)
	}
	next := money.Format6(current.Add(amount))

	if affected, err := queries.UpdateUserBalance(ctx, sqlc.UpdateUserBalanceParams{AvailableBalance: next, UserID: int64(id)}); err != nil {
		return nil, mapError(err)
	} else if affected == 0 {
		return nil, store.ErrNotFound
	}

	var orderID any
	if in.RelatedOrderID != "" {
		orderID = in.RelatedOrderID
	}
	if _, err := queries.CreateBalanceTransaction(ctx, sqlc.CreateBalanceTransactionParams{
		UserID:         int64(id),
		TxType:         "recharge",
		Amount:         money.Format6(amount),
		BalanceAfter:   next,
		RelatedOrderID: orderID,
		Description:    in.Description,
	}); err != nil {
		return nil, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, mapError(err)
	}
	return map[string]any{"balance_after": next}, nil
}

func (s *Store) GetUserBalance(id int) (map[string]any, error) {
	row, err := s.queries.GetUserBalanceText(context.Background(), int64(id))
	if err != nil {
		return nil, mapError(err)
	}
	return balanceDTO(textValue(row.AvailableBalance), textValue(row.FrozenBalance)), nil
}

func (s *Store) ListBalanceTransactions(userID, page, pageSize int) (domain.ListResponse, error) {
	ctx := context.Background()
	if _, err := s.queries.GetUser(ctx, int64(userID)); err != nil {
		return domain.ListResponse{}, mapError(err)
	}

	limit, offset := limitOffset(page, pageSize)
	rows, err := s.queries.ListBalanceTransactions(ctx, sqlc.ListBalanceTransactionsParams{UserID: int64(userID), Limit: limit, Offset: offset})
	if err != nil {
		return domain.ListResponse{}, mapError(err)
	}
	total, err := s.queries.CountBalanceTransactions(ctx, int64(userID))
	if err != nil {
		return domain.ListResponse{}, mapError(err)
	}

	list := []any{}
	for _, row := range rows {
		list = append(list, map[string]any{
			"id":            int(row.ID),
			"tx_type":       row.TxType,
			"amount":        row.Amount,
			"balance_after": row.BalanceAfter,
			"created_at":    row.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return domain.ListResponse{List: list, Total: int(total)}, nil
}

func (s *Store) ListUserKeys(userID, page, pageSize int) (domain.ListResponse, error) {
	ctx := context.Background()
	if _, err := s.queries.GetUser(ctx, int64(userID)); err != nil {
		return domain.ListResponse{}, mapError(err)
	}

	limit, offset := limitOffset(page, pageSize)
	rows, err := s.queries.ListUserKeys(ctx, sqlc.ListUserKeysParams{UserID: int64(userID), Limit: limit, Offset: offset})
	if err != nil {
		return domain.ListResponse{}, mapError(err)
	}
	total, err := s.queries.CountUserKeys(ctx, int64(userID))
	if err != nil {
		return domain.ListResponse{}, mapError(err)
	}

	list := []any{}
	for _, row := range rows {
		list = append(list, keyDTO(row.ID, row.UserID, row.KeyName, row.Prefix, row.IsActive, row.LastUsedAt, row.ExpiresAt))
	}
	return domain.ListResponse{List: list, Total: int(total)}, nil
}

func (s *Store) ListKeys(page, pageSize int) (domain.ListResponse, error) {
	ctx := context.Background()
	limit, offset := limitOffset(page, pageSize)

	rows, err := s.queries.ListKeys(ctx, sqlc.ListKeysParams{Limit: limit, Offset: offset})
	if err != nil {
		return domain.ListResponse{}, mapError(err)
	}
	total, err := s.queries.CountKeys(ctx)
	if err != nil {
		return domain.ListResponse{}, mapError(err)
	}

	list := []any{}
	for _, row := range rows {
		list = append(list, keyDTO(row.ID, row.UserID, row.KeyName, row.Prefix, row.IsActive, row.LastUsedAt, row.ExpiresAt))
	}
	return domain.ListResponse{List: list, Total: int(total)}, nil
}

func (s *Store) CreateKey(userID int, in domain.KeyInput) (map[string]any, error) {
	ctx := context.Background()
	if _, err := s.queries.GetUser(ctx, int64(userID)); err != nil {
		return nil, mapError(err)
	}

	keyName := in.KeyName
	if keyName == "" {
		keyName = "default"
	}
	prefix := in.Prefix
	if prefix == "" {
		prefix = "sk-"
	}
	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	fullKey, err := crypto.GenerateGatewayKey(prefix)
	if err != nil {
		return nil, err
	}

	id, err := s.queries.CreateKey(ctx, sqlc.CreateKeyParams{
		UserID:             int64(userID),
		KeyName:            keyName,
		Prefix:             prefix,
		KeyHash:            crypto.HashKey(fullKey),
		Permissions:        defaultJSON(in.Permissions, defaultPermissions),
		RateLimitOverrides: rawJSON(in.RateLimitOverrides),
		ExpiresAt:          in.ExpiresAt,
		IsActive:           isActive,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return map[string]any{"id": int(id), "full_key": fullKey}, nil
}

func (s *Store) UpdateKey(userID, keyID int, in domain.KeyUpdateInput) (map[string]any, error) {
	if in.IsActive == nil {
		return nil, fmt.Errorf("%w: is_active is required", store.ErrInvalid)
	}
	ctx := context.Background()
	affected, err := s.queries.UpdateKeyActive(ctx, sqlc.UpdateKeyActiveParams{IsActive: *in.IsActive, ID: int64(keyID), UserID: int64(userID)})
	if err != nil {
		return nil, mapError(err)
	}
	if affected == 0 {
		return nil, store.ErrNotFound
	}

	row, err := s.queries.GetKey(ctx, sqlc.GetKeyParams{ID: int64(keyID), UserID: int64(userID)})
	if err != nil {
		return nil, mapError(err)
	}
	return keyDTO(row.ID, row.UserID, row.KeyName, row.Prefix, row.IsActive, row.LastUsedAt, row.ExpiresAt), nil
}

func (s *Store) DeleteKey(userID, keyID int) error {
	affected, err := s.queries.DeleteKey(context.Background(), sqlc.DeleteKeyParams{ID: int64(keyID), UserID: int64(userID)})
	if err != nil {
		return mapError(err)
	}
	if affected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *Store) ResetKey(userID, keyID int) (map[string]any, error) {
	ctx := context.Background()
	row, err := s.queries.GetKey(ctx, sqlc.GetKeyParams{ID: int64(keyID), UserID: int64(userID)})
	if err != nil {
		return nil, mapError(err)
	}

	fullKey, err := crypto.GenerateGatewayKey(row.Prefix)
	if err != nil {
		return nil, err
	}
	affected, err := s.queries.UpdateKeySecret(ctx, sqlc.UpdateKeySecretParams{KeyHash: crypto.HashKey(fullKey), Prefix: row.Prefix, ID: int64(keyID), UserID: int64(userID)})
	if err != nil {
		return nil, mapError(err)
	}
	if affected == 0 {
		return nil, store.ErrNotFound
	}
	return map[string]any{"full_key": fullKey}, nil
}

func (s *Store) getUserDTO(id int64) (map[string]any, error) {
	ctx := context.Background()
	user, err := s.queries.GetUser(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	balance, err := s.queries.GetUserBalanceText(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return userDTO(user.ID, user.Nickname, user.UserGroup, user.Status, textValue(balance.AvailableBalance), textValue(balance.FrozenBalance)), nil
}

func userDTO(id int64, nickname, userGroup, status, available, frozen string) map[string]any {
	return map[string]any{
		"id":         int(id),
		"nickname":   nickname,
		"user_group": userGroup,
		"status":     status,
		"balance":    balanceDTO(available, frozen),
	}
}

func balanceDTO(available, frozen string) map[string]any {
	return map[string]any{"available_balance": available, "frozen_balance": frozen}
}

func keyDTO(id, userID int64, keyName, prefix string, isActive bool, lastUsedAt, expiresAt pgtype.Timestamptz) map[string]any {
	return map[string]any{
		"id":           int(id),
		"user_id":      int(userID),
		"key_name":     keyName,
		"prefix":       prefix,
		"is_active":    isActive,
		"last_used_at": optionalTimestamp(lastUsedAt),
		"expires_at":   optionalTimestamp(expiresAt),
	}
}

func limitOffset(page, pageSize int) (int32, int32) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	return int32(pageSize), int32((page - 1) * pageSize)
}

func defaultJSON(value json.RawMessage, fallback string) []byte {
	trimmed := trimJSON(value)
	if trimmed == "" {
		return []byte(fallback)
	}
	return []byte(trimmed)
}

func rawJSON(value json.RawMessage) []byte {
	trimmed := trimJSON(value)
	if trimmed == "" {
		return nil
	}
	return []byte(trimmed)
}

func trimJSON(value json.RawMessage) string {
	trimmed := strings.TrimSpace(string(value))
	if trimmed == "null" {
		return ""
	}
	return trimmed
}
