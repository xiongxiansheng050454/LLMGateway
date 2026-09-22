package storefake

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	domain "LLMGateway/server/internal/accounts"
	"LLMGateway/server/internal/store"
)

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// --- Port reads ---

func (s *Store) ListUsers(page, pageSize int) (domain.ListResponse[domain.UserDTO], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]int, 0, len(s.users))
	for id := range s.users {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	start, end := pageBounds(len(ids), page, pageSize)
	list := []domain.UserDTO{}
	for _, id := range ids[start:end] {
		list = append(list, s.userDTO(s.users[id]))
	}
	return domain.ListResponse[domain.UserDTO]{List: list, Total: len(ids)}, nil
}

func (s *Store) GetUserBalance(id int) (domain.BalanceDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return domain.BalanceDTO{}, store.ErrNotFound
	}
	return balanceDTO(user.AvailableBalance, user.FrozenBalance), nil
}

func (s *Store) ListBalanceTransactions(userID, page, pageSize int) (domain.ListResponse[domain.BalanceTransactionDTO], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return domain.ListResponse[domain.BalanceTransactionDTO]{}, store.ErrNotFound
	}

	rows := s.transactions[userID]
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID > rows[j].ID })
	start, end := pageBounds(len(rows), page, pageSize)
	list := []domain.BalanceTransactionDTO{}
	for _, tx := range rows[start:end] {
		list = append(list, domain.BalanceTransactionDTO{ID: tx.ID, TxType: tx.TxType, Amount: tx.Amount, BalanceAfter: tx.BalanceAfter, CreatedAt: tx.CreatedAt})
	}
	return domain.ListResponse[domain.BalanceTransactionDTO]{List: list, Total: len(rows)}, nil
}

func (s *Store) ListUserKeys(userID, page, pageSize int) (domain.ListResponse[domain.ClientKeyDTO], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return domain.ListResponse[domain.ClientKeyDTO]{}, store.ErrNotFound
	}
	keys := s.sortedKeysLocked(userID)
	start, end := pageBounds(len(keys), page, pageSize)
	list := []domain.ClientKeyDTO{}
	for _, key := range keys[start:end] {
		list = append(list, keyDTO(key))
	}
	return domain.ListResponse[domain.ClientKeyDTO]{List: list, Total: len(keys)}, nil
}

func (s *Store) ListKeys(page, pageSize int) (domain.ListResponse[domain.ClientKeyDTO], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := s.sortedKeysLocked(0)
	start, end := pageBounds(len(keys), page, pageSize)
	list := []domain.ClientKeyDTO{}
	for _, key := range keys[start:end] {
		list = append(list, keyDTO(key))
	}
	return domain.ListResponse[domain.ClientKeyDTO]{List: list, Total: len(keys)}, nil
}

func (s *Store) AuthenticateKey(keyHash string) (*domain.AuthContext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, key := range s.keys {
		if key.keyHash != keyHash {
			continue
		}
		user, ok := s.users[key.userID]
		if !ok {
			return nil, store.ErrNotFound
		}
		return &domain.AuthContext{
			KeyID:              key.id,
			UserID:             key.userID,
			KeyName:            key.keyName,
			KeyActive:          key.isActive,
			ExpiresAt:          key.expiresAt,
			Permissions:        key.permissions,
			RateLimitOverrides: key.rateLimitOverrides,
			UserStatus:         user.Status,
			AvailableBalance:   user.AvailableBalance,
			FrozenBalance:      user.FrozenBalance,
		}, nil
	}
	return nil, store.ErrNotFound
}

func (s *Store) UpdateKeyLastUsed(keyID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.keys[keyID]
	if !ok {
		return store.ErrNotFound
	}
	now := nowRFC3339()
	key.lastUsedAt = &now
	return nil
}

// --- Tx primitives ---

// accountsRunner runs the callback while holding the store mutex so the
// primitives observe a consistent snapshot, mirroring a database transaction.
type accountsRunner struct {
	store *Store
}

func (s *Store) AccountsTx() domain.TxManager { return accountsRunner{store: s} }

func (r accountsRunner) InTx(_ context.Context, fn func(domain.Tx) error) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	return fn(&accountsTx{s: r.store})
}

type accountsTx struct {
	s *Store
}

func (t *accountsTx) GetUser(id int) (domain.User, error) {
	user, ok := t.s.users[id]
	if !ok {
		return domain.User{}, store.ErrNotFound
	}
	return *user, nil
}

func (t *accountsTx) InsertUser(nickname, group, status string) (domain.User, error) {
	user := &domain.User{ID: t.s.nextUserID, Nickname: nickname, UserGroup: group, Status: status, AvailableBalance: "0.000000", FrozenBalance: "0.000000"}
	t.s.nextUserID++
	t.s.users[user.ID] = user
	return *user, nil
}

func (t *accountsTx) InsertUserBalance(int) error { return nil }

func (t *accountsTx) UpdateUser(id int, nickname, group string) (domain.User, bool, error) {
	user, ok := t.s.users[id]
	if !ok {
		return domain.User{}, false, nil
	}
	user.Nickname = nickname
	user.UserGroup = group
	return *user, true, nil
}

func (t *accountsTx) UpdateUserStatus(id int, status string) (domain.User, bool, error) {
	user, ok := t.s.users[id]
	if !ok {
		return domain.User{}, false, nil
	}
	user.Status = status
	return *user, true, nil
}

func (t *accountsTx) DeleteUser(id int) (bool, error) {
	if _, ok := t.s.users[id]; !ok {
		return false, nil
	}
	t.s.cleanupQuotaLocked(id, 0)
	delete(t.s.users, id)
	delete(t.s.transactions, id)
	for keyID, key := range t.s.keys {
		if key.userID == id {
			delete(t.s.keys, keyID)
		}
	}
	for order, tx := range t.s.orders {
		if tx.ID != 0 && strings.HasPrefix(order, fmt.Sprintf("%d:", id)) {
			delete(t.s.orders, order)
		}
	}
	return true, nil
}

func (t *accountsTx) LockUserBalance(userID int) error {
	if _, ok := t.s.users[userID]; !ok {
		return store.ErrNotFound
	}
	return nil
}

func (t *accountsTx) GetUserBalanceText(userID int) (string, error) {
	user, ok := t.s.users[userID]
	if !ok {
		return "", store.ErrNotFound
	}
	return user.AvailableBalance, nil
}

func (t *accountsTx) UpdateUserBalance(userID int, available string) (bool, error) {
	user, ok := t.s.users[userID]
	if !ok {
		return false, nil
	}
	user.AvailableBalance = available
	return true, nil
}

func (t *accountsTx) InsertBalanceTransaction(in domain.BalanceTransactionInput) error {
	tx := domain.BalanceTransaction{
		ID:           t.s.nextTxID,
		TxType:       in.TxType,
		Amount:       in.Amount,
		BalanceAfter: in.BalanceAfter,
		Description:  in.Description,
		CreatedAt:    nowRFC3339(),
	}
	t.s.nextTxID++
	t.s.transactions[in.UserID] = append(t.s.transactions[in.UserID], tx)
	if in.RelatedOrderID != "" {
		t.s.orders[orderKey(in.UserID, in.RelatedOrderID)] = tx
	}
	return nil
}

func (t *accountsTx) GetBalanceTransactionByOrder(userID int, orderID string) (string, bool, error) {
	existing, ok := t.s.orders[orderKey(userID, orderID)]
	if !ok {
		return "", false, nil
	}
	return existing.BalanceAfter, true, nil
}

func (t *accountsTx) GetKey(keyID, userID int) (domain.ClientKey, error) {
	key, ok := t.s.keys[keyID]
	if !ok || key.userID != userID {
		return domain.ClientKey{}, store.ErrNotFound
	}
	return clientKey(key), nil
}

func (t *accountsTx) InsertKey(in domain.KeyInsert) (int, error) {
	key := &memoryKey{
		id:                 t.s.nextKeyID,
		userID:             in.UserID,
		keyName:            in.KeyName,
		prefix:             in.Prefix,
		keyHash:            in.KeyHash,
		permissions:        canonicalJSON(normalizeJSON(in.Permissions, defaultPermissions)),
		rateLimitOverrides: canonicalJSON(normalizeJSON(in.RateLimitOverrides, "")),
		isActive:           in.IsActive,
		expiresAt:          normalizeTimestampPtr(in.ExpiresAt),
	}
	t.s.nextKeyID++
	t.s.keys[key.id] = key
	return key.id, nil
}

func (t *accountsTx) UpdateKeyActive(keyID, userID int, active bool) (domain.ClientKey, bool, error) {
	key, ok := t.s.keys[keyID]
	if !ok || key.userID != userID {
		return domain.ClientKey{}, false, nil
	}
	key.isActive = active
	return clientKey(key), true, nil
}

func (t *accountsTx) UpdateKeySecret(keyID, userID int, keyHash, prefix string) (bool, error) {
	key, ok := t.s.keys[keyID]
	if !ok || key.userID != userID {
		return false, nil
	}
	key.keyHash = keyHash
	key.prefix = prefix
	return true, nil
}

func (t *accountsTx) DeleteKey(keyID, userID int) (bool, error) {
	key, ok := t.s.keys[keyID]
	if !ok || key.userID != userID {
		return false, nil
	}
	t.s.cleanupQuotaLocked(userID, keyID)
	delete(t.s.keys, keyID)
	return true, nil
}

func (t *accountsTx) DeleteQuotaReservationsForUser(userID int) error {
	t.s.cleanupQuotaLocked(userID, 0)
	return nil
}

func (t *accountsTx) DeleteQuotaReservationsForKey(userID, keyID int) error {
	t.s.cleanupQuotaLocked(userID, keyID)
	return nil
}

// --- mappings ---

func (s *Store) userDTO(user *domain.User) domain.UserDTO {
	return domain.UserDTO{ID: user.ID, Nickname: user.Nickname, UserGroup: user.UserGroup, Status: user.Status, Balance: balanceDTO(user.AvailableBalance, user.FrozenBalance)}
}

func balanceDTO(available, frozen string) domain.BalanceDTO {
	return domain.BalanceDTO{AvailableBalance: available, FrozenBalance: frozen}
}

func keyDTO(key *memoryKey) domain.ClientKeyDTO {
	return domain.ClientKeyDTO(clientKey(key))
}

func clientKey(key *memoryKey) domain.ClientKey {
	return domain.ClientKey{ID: key.id, UserID: key.userID, KeyName: key.keyName, Prefix: key.prefix, IsActive: key.isActive, LastUsedAt: key.lastUsedAt, ExpiresAt: key.expiresAt}
}

func (s *Store) sortedKeysLocked(userID int) []*memoryKey {
	keys := []*memoryKey{}
	for _, key := range s.keys {
		if userID == 0 || key.userID == userID {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].id < keys[j].id })
	return keys
}

const defaultPermissions = `{"models":["*"]}`

func normalizeJSON(value json.RawMessage, fallback string) json.RawMessage {
	trimmed := strings.TrimSpace(string(value))
	if trimmed == "" || trimmed == "null" {
		if fallback == "" {
			return nil
		}
		return json.RawMessage(fallback)
	}
	return value
}

// normalizeTimestampPtr parses an RFC3339 timestamp and returns it in UTC so
// the fake store matches the PostgreSQL timestamptz output. Unparseable
// values are preserved as-is.
func normalizeTimestampPtr(value string) *string {
	if value == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		formatted := parsed.UTC().Format(time.RFC3339)
		return &formatted
	}
	return &value
}

func orderKey(userID int, orderID string) string {
	return fmt.Sprintf("%d:%s", userID, orderID)
}

func pageBounds(total, page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}
