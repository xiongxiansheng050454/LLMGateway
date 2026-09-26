package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	domain "LLMGateway/server/internal/accounts"
	"LLMGateway/server/internal/db/sqlc"
	"LLMGateway/server/internal/store"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- Store reads ---

func (s *Store) ListUsers(ctx context.Context, page, pageSize int) (domain.ListResponse[domain.UserDTO], error) {
	limit, offset := limitOffset(page, pageSize)

	rows, err := s.queries.ListUsers(ctx, sqlc.ListUsersParams{Limit: limit, Offset: offset})
	if err != nil {
		return domain.ListResponse[domain.UserDTO]{}, mapError(err)
	}
	total, err := s.queries.CountUsers(ctx)
	if err != nil {
		return domain.ListResponse[domain.UserDTO]{}, mapError(err)
	}

	list := []domain.UserDTO{}
	for _, row := range rows {
		list = append(list, userDTO(row.ID, row.Nickname, row.UserGroup, row.Status, textValue(row.AvailableBalance), textValue(row.FrozenBalance)))
	}
	return domain.ListResponse[domain.UserDTO]{List: list, Total: int(total)}, nil
}

func (s *Store) GetUserBalance(ctx context.Context, id int) (domain.BalanceDTO, error) {
	row, err := s.queries.GetUserBalanceText(ctx, int64(id))
	if err != nil {
		return domain.BalanceDTO{}, mapError(err)
	}
	return balanceDTO(textValue(row.AvailableBalance), textValue(row.FrozenBalance)), nil
}

func (s *Store) ListBalanceTransactions(ctx context.Context, userID, page, pageSize int) (domain.ListResponse[domain.BalanceTransactionDTO], error) {
	if _, err := s.queries.GetUser(ctx, int64(userID)); err != nil {
		return domain.ListResponse[domain.BalanceTransactionDTO]{}, mapError(err)
	}

	limit, offset := limitOffset(page, pageSize)
	rows, err := s.queries.ListBalanceTransactions(ctx, sqlc.ListBalanceTransactionsParams{UserID: int64(userID), Limit: limit, Offset: offset})
	if err != nil {
		return domain.ListResponse[domain.BalanceTransactionDTO]{}, mapError(err)
	}
	total, err := s.queries.CountBalanceTransactions(ctx, int64(userID))
	if err != nil {
		return domain.ListResponse[domain.BalanceTransactionDTO]{}, mapError(err)
	}

	list := []domain.BalanceTransactionDTO{}
	for _, row := range rows {
		list = append(list, domain.BalanceTransactionDTO{ID: int(row.ID), TxType: row.TxType, Amount: row.Amount, BalanceAfter: row.BalanceAfter, CreatedAt: row.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z07:00")})
	}
	return domain.ListResponse[domain.BalanceTransactionDTO]{List: list, Total: int(total)}, nil
}

func (s *Store) ListUserKeys(ctx context.Context, userID, page, pageSize int) (domain.ListResponse[domain.ClientKeyDTO], error) {
	if _, err := s.queries.GetUser(ctx, int64(userID)); err != nil {
		return domain.ListResponse[domain.ClientKeyDTO]{}, mapError(err)
	}

	limit, offset := limitOffset(page, pageSize)
	rows, err := s.queries.ListUserKeys(ctx, sqlc.ListUserKeysParams{UserID: int64(userID), Limit: limit, Offset: offset})
	if err != nil {
		return domain.ListResponse[domain.ClientKeyDTO]{}, mapError(err)
	}
	total, err := s.queries.CountUserKeys(ctx, int64(userID))
	if err != nil {
		return domain.ListResponse[domain.ClientKeyDTO]{}, mapError(err)
	}

	list := []domain.ClientKeyDTO{}
	for _, row := range rows {
		list = append(list, keyDTO(row.ID, row.UserID, row.KeyName, row.Prefix, row.IsActive, row.LastUsedAt, row.ExpiresAt))
	}
	return domain.ListResponse[domain.ClientKeyDTO]{List: list, Total: int(total)}, nil
}

func (s *Store) ListKeys(ctx context.Context, page, pageSize int) (domain.ListResponse[domain.ClientKeyDTO], error) {
	limit, offset := limitOffset(page, pageSize)

	rows, err := s.queries.ListKeys(ctx, sqlc.ListKeysParams{Limit: limit, Offset: offset})
	if err != nil {
		return domain.ListResponse[domain.ClientKeyDTO]{}, mapError(err)
	}
	total, err := s.queries.CountKeys(ctx)
	if err != nil {
		return domain.ListResponse[domain.ClientKeyDTO]{}, mapError(err)
	}

	list := []domain.ClientKeyDTO{}
	for _, row := range rows {
		list = append(list, keyDTO(row.ID, row.UserID, row.KeyName, row.Prefix, row.IsActive, row.LastUsedAt, row.ExpiresAt))
	}
	return domain.ListResponse[domain.ClientKeyDTO]{List: list, Total: int(total)}, nil
}

func (s *Store) AuthenticateKey(ctx context.Context, keyHash string) (*domain.AuthContext, error) {
	row, err := s.queries.GetAuthContextByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, mapError(err)
	}
	return &domain.AuthContext{
		KeyID:              int(row.KeyID),
		UserID:             int(row.UserID),
		KeyName:            row.KeyName,
		KeyActive:          row.KeyActive,
		ExpiresAt:          optionalTimestamp(row.ExpiresAt),
		Permissions:        canonicalJSON(json.RawMessage(row.Permissions)),
		RateLimitOverrides: canonicalJSON(json.RawMessage(row.RateLimitOverrides)),
		UserStatus:         row.UserStatus,
		AvailableBalance:   textValue(row.AvailableBalance),
		FrozenBalance:      textValue(row.FrozenBalance),
	}, nil
}

func (s *Store) UpdateKeyLastUsed(ctx context.Context, keyID int) error {
	affected, err := s.queries.UpdateKeyLastUsed(ctx, int64(keyID))
	if err != nil {
		return mapError(err)
	}
	if affected == 0 {
		return store.ErrNotFound
	}
	return nil
}

// --- Tx primitives ---

func (t *Tx) GetUser(id int) (domain.User, error) {
	row, err := t.queries.GetUser(t.ctx, int64(id))
	if err != nil {
		return domain.User{}, mapError(err)
	}
	balance, err := t.queries.GetUserBalanceText(t.ctx, int64(id))
	if err != nil {
		return domain.User{}, mapError(err)
	}
	return accountUser(row.ID, row.Nickname, row.UserGroup, row.Status, textValue(balance.AvailableBalance), textValue(balance.FrozenBalance)), nil
}

func (t *Tx) InsertUser(nickname, group, status string) (domain.User, error) {
	id, err := t.queries.CreateUser(t.ctx, sqlc.CreateUserParams{Nickname: nickname, UserGroup: group, Status: status})
	if err != nil {
		return domain.User{}, mapError(err)
	}
	return accountUser(id, nickname, group, status, "0.000000", "0.000000"), nil
}

func (t *Tx) InsertUserBalance(userID int) error {
	return mapError(t.queries.CreateUserBalance(t.ctx, int64(userID)))
}

func (t *Tx) UpdateUser(id int, nickname, group string) (domain.User, bool, error) {
	affected, err := t.queries.UpdateUser(t.ctx, sqlc.UpdateUserParams{Nickname: nickname, UserGroup: group, ID: int64(id)})
	if err != nil {
		return domain.User{}, false, mapError(err)
	}
	if affected == 0 {
		return domain.User{}, false, nil
	}
	user, err := t.GetUser(id)
	if err != nil {
		return domain.User{}, false, err
	}
	return user, true, nil
}

func (t *Tx) UpdateUserStatus(id int, status string) (domain.User, bool, error) {
	affected, err := t.queries.UpdateUserStatus(t.ctx, sqlc.UpdateUserStatusParams{Status: status, ID: int64(id)})
	if err != nil {
		return domain.User{}, false, mapError(err)
	}
	if affected == 0 {
		return domain.User{}, false, nil
	}
	user, err := t.GetUser(id)
	if err != nil {
		return domain.User{}, false, err
	}
	return user, true, nil
}

func (t *Tx) DeleteUser(id int) (bool, error) {
	affected, err := t.queries.DeleteUser(t.ctx, int64(id))
	if err != nil {
		return false, mapError(err)
	}
	return affected > 0, nil
}

func (t *Tx) LockUserBalance(userID int) error {
	_, err := t.queries.LockUserBalance(t.ctx, int64(userID))
	return mapError(err)
}

func (t *Tx) GetUserBalanceText(userID int) (string, error) {
	row, err := t.queries.GetUserBalanceText(t.ctx, int64(userID))
	if err != nil {
		return "", mapError(err)
	}
	return textValue(row.AvailableBalance), nil
}

func (t *Tx) UpdateUserBalance(userID int, available string) (bool, error) {
	affected, err := t.queries.UpdateUserBalance(t.ctx, sqlc.UpdateUserBalanceParams{AvailableBalance: available, UserID: int64(userID)})
	if err != nil {
		return false, mapError(err)
	}
	return affected > 0, nil
}

func (t *Tx) InsertBalanceTransaction(in domain.BalanceTransactionInput) error {
	var orderID any
	if in.RelatedOrderID != "" {
		orderID = in.RelatedOrderID
	}
	_, err := t.queries.CreateBalanceTransaction(t.ctx, sqlc.CreateBalanceTransactionParams{
		UserID:         int64(in.UserID),
		TxType:         in.TxType,
		Amount:         in.Amount,
		BalanceAfter:   in.BalanceAfter,
		RelatedOrderID: orderID,
		Description:    in.Description,
	})
	return mapError(err)
}

func (t *Tx) GetBalanceTransactionByOrder(userID int, orderID string) (string, bool, error) {
	row, err := t.queries.GetBalanceTransactionByOrder(t.ctx, sqlc.GetBalanceTransactionByOrderParams{
		UserID:         int64(userID),
		RelatedOrderID: pgtype.Text{String: orderID, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, mapError(err)
	}
	return row.BalanceAfter, true, nil
}

func (t *Tx) GetKey(keyID, userID int) (domain.ClientKey, error) {
	row, err := t.queries.GetKey(t.ctx, sqlc.GetKeyParams{ID: int64(keyID), UserID: int64(userID)})
	if err != nil {
		return domain.ClientKey{}, mapError(err)
	}
	return domain.ClientKey{
		ID:         int(row.ID),
		UserID:     int(row.UserID),
		KeyName:    row.KeyName,
		Prefix:     row.Prefix,
		IsActive:   row.IsActive,
		LastUsedAt: optionalTimestamp(row.LastUsedAt),
		ExpiresAt:  optionalTimestamp(row.ExpiresAt),
	}, nil
}

func (t *Tx) InsertKey(in domain.KeyInsert) (int, error) {
	id, err := t.queries.CreateKey(t.ctx, sqlc.CreateKeyParams{
		UserID:             int64(in.UserID),
		KeyName:            in.KeyName,
		Prefix:             in.Prefix,
		KeyHash:            in.KeyHash,
		Permissions:        []byte(in.Permissions),
		RateLimitOverrides: rawJSON(in.RateLimitOverrides),
		ExpiresAt:          in.ExpiresAt,
		IsActive:           in.IsActive,
	})
	if err != nil {
		return 0, mapError(err)
	}
	return int(id), nil
}

func (t *Tx) UpdateKeyActive(keyID, userID int, active bool) (domain.ClientKey, bool, error) {
	affected, err := t.queries.UpdateKeyActive(t.ctx, sqlc.UpdateKeyActiveParams{IsActive: active, ID: int64(keyID), UserID: int64(userID)})
	if err != nil {
		return domain.ClientKey{}, false, mapError(err)
	}
	if affected == 0 {
		return domain.ClientKey{}, false, nil
	}
	key, err := t.GetKey(keyID, userID)
	if err != nil {
		return domain.ClientKey{}, false, err
	}
	return key, true, nil
}

func (t *Tx) UpdateKeySecret(keyID, userID int, keyHash, prefix string) (bool, error) {
	affected, err := t.queries.UpdateKeySecret(t.ctx, sqlc.UpdateKeySecretParams{KeyHash: keyHash, Prefix: prefix, ID: int64(keyID), UserID: int64(userID)})
	if err != nil {
		return false, mapError(err)
	}
	return affected > 0, nil
}

func (t *Tx) DeleteKey(keyID, userID int) (bool, error) {
	affected, err := t.queries.DeleteKey(t.ctx, sqlc.DeleteKeyParams{ID: int64(keyID), UserID: int64(userID)})
	if err != nil {
		return false, mapError(err)
	}
	return affected > 0, nil
}

func (t *Tx) DeleteQuotaReservationsForUser(userID int) error {
	return cleanupQuotaReservationsTx(t.ctx, t.tx, userID, 0)
}

func (t *Tx) DeleteQuotaReservationsForKey(userID, keyID int) error {
	return cleanupQuotaReservationsTx(t.ctx, t.tx, userID, keyID)
}

// --- mappings ---

func accountUser(id int64, nickname, userGroup, status, available, frozen string) domain.User {
	return domain.User{ID: int(id), Nickname: nickname, UserGroup: userGroup, Status: status, AvailableBalance: available, FrozenBalance: frozen}
}

func userDTO(id int64, nickname, userGroup, status, available, frozen string) domain.UserDTO {
	return domain.UserDTO{ID: int(id), Nickname: nickname, UserGroup: userGroup, Status: status, Balance: balanceDTO(available, frozen)}
}

func balanceDTO(available, frozen string) domain.BalanceDTO {
	return domain.BalanceDTO{AvailableBalance: available, FrozenBalance: frozen}
}

func keyDTO(id, userID int64, keyName, prefix string, isActive bool, lastUsedAt, expiresAt pgtype.Timestamptz) domain.ClientKeyDTO {
	return domain.ClientKeyDTO{ID: int(id), UserID: int(userID), KeyName: keyName, Prefix: prefix, IsActive: isActive, LastUsedAt: optionalTimestamp(lastUsedAt), ExpiresAt: optionalTimestamp(expiresAt)}
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
