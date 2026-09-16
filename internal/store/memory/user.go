package memory

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"LLMGateway/internal/crypto"
	"LLMGateway/internal/domain"
	"LLMGateway/internal/money"
	"LLMGateway/internal/store"
)

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func (s *Store) ListUsers(page, pageSize int) (domain.ListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]int, 0, len(s.users))
	for id := range s.users {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	start, end := pageBounds(len(ids), page, pageSize)
	list := []any{}
	for _, id := range ids[start:end] {
		list = append(list, s.userDTO(s.users[id]))
	}
	return domain.ListResponse{List: list, Total: len(ids)}, nil
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

	s.mu.Lock()
	defer s.mu.Unlock()
	user := &domain.User{ID: s.nextUserID, Nickname: in.Nickname, UserGroup: in.UserGroup, Status: in.Status, AvailableBalance: "0.000000", FrozenBalance: "0.000000"}
	s.nextUserID++
	s.users[user.ID] = user
	return s.userDTO(user), nil
}

func (s *Store) UpdateUser(id int, in domain.UserInput) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	if in.UserGroup != "" {
		user.UserGroup = in.UserGroup
	}
	user.Nickname = in.Nickname
	return s.userDTO(user), nil
}

func (s *Store) UpdateUserStatus(id int, status string) (map[string]any, error) {
	if status != "active" && status != "suspended" {
		return nil, fmt.Errorf("%w: invalid status", store.ErrInvalid)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	user.Status = status
	return s.userDTO(user), nil
}

func (s *Store) DeleteUser(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[id]; !ok {
		return store.ErrNotFound
	}
	delete(s.users, id)
	delete(s.transactions, id)
	for keyID, key := range s.keys {
		if key.userID == id {
			delete(s.keys, keyID)
		}
	}
	for order, tx := range s.orders {
		if tx.ID != 0 && strings.HasPrefix(order, fmt.Sprintf("%d:", id)) {
			delete(s.orders, order)
		}
	}
	return nil
}

func (s *Store) RechargeUser(id int, in domain.RechargeInput) (map[string]any, error) {
	amount, err := money.Parse6(in.Amount)
	if err != nil || amount.Cmp(0) <= 0 {
		return nil, fmt.Errorf("%w: invalid amount", store.ErrInvalid)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return nil, store.ErrNotFound
	}

	if in.RelatedOrderID != "" {
		if existing, ok := s.orders[orderKey(id, in.RelatedOrderID)]; ok {
			return map[string]any{"balance_after": existing.BalanceAfter}, nil
		}
	}

	current, err := money.Parse6(user.AvailableBalance)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid balance", store.ErrInvalid)
	}
	next := current.Add(amount)
	user.AvailableBalance = money.Format6(next)

	tx := domain.BalanceTransaction{
		ID:           s.nextTxID,
		TxType:       "recharge",
		Amount:       money.Format6(amount),
		BalanceAfter: user.AvailableBalance,
		CreatedAt:    nowRFC3339(),
	}
	s.nextTxID++
	s.transactions[id] = append(s.transactions[id], tx)
	if in.RelatedOrderID != "" {
		s.orders[orderKey(id, in.RelatedOrderID)] = tx
	}
	return map[string]any{"balance_after": user.AvailableBalance}, nil
}

func (s *Store) GetUserBalance(id int) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	return balanceDTO(user.AvailableBalance, user.FrozenBalance), nil
}

func (s *Store) ListBalanceTransactions(userID, page, pageSize int) (domain.ListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return domain.ListResponse{}, store.ErrNotFound
	}

	rows := s.transactions[userID]
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID > rows[j].ID })
	start, end := pageBounds(len(rows), page, pageSize)
	list := []any{}
	for _, tx := range rows[start:end] {
		list = append(list, map[string]any{"id": tx.ID, "tx_type": tx.TxType, "amount": tx.Amount, "balance_after": tx.BalanceAfter, "created_at": tx.CreatedAt})
	}
	return domain.ListResponse{List: list, Total: len(rows)}, nil
}

func (s *Store) ListUserKeys(userID, page, pageSize int) (domain.ListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return domain.ListResponse{}, store.ErrNotFound
	}
	keys := s.sortedKeysLocked(userID)
	start, end := pageBounds(len(keys), page, pageSize)
	list := []any{}
	for _, key := range keys[start:end] {
		list = append(list, keyDTO(key))
	}
	return domain.ListResponse{List: list, Total: len(keys)}, nil
}

func (s *Store) ListKeys(page, pageSize int) (domain.ListResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := s.sortedKeysLocked(0)
	start, end := pageBounds(len(keys), page, pageSize)
	list := []any{}
	for _, key := range keys[start:end] {
		list = append(list, keyDTO(key))
	}
	return domain.ListResponse{List: list, Total: len(keys)}, nil
}

func (s *Store) CreateKey(userID int, in domain.KeyInput) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return nil, store.ErrNotFound
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

	key := &memoryKey{
		id:                 s.nextKeyID,
		userID:             userID,
		keyName:            keyName,
		prefix:             prefix,
		keyHash:            crypto.HashKey(fullKey),
		permissions:        normalizeJSON(in.Permissions, defaultPermissions),
		rateLimitOverrides: normalizeJSON(in.RateLimitOverrides, ""),
		isActive:           isActive,
		expiresAt:          normalizeTimestampPtr(in.ExpiresAt),
	}
	s.nextKeyID++
	s.keys[key.id] = key
	return map[string]any{"id": key.id, "full_key": fullKey}, nil
}

func (s *Store) UpdateKey(userID, keyID int, in domain.KeyUpdateInput) (map[string]any, error) {
	if in.IsActive == nil {
		return nil, fmt.Errorf("%w: is_active is required", store.ErrInvalid)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.keys[keyID]
	if !ok || key.userID != userID {
		return nil, store.ErrNotFound
	}
	key.isActive = *in.IsActive
	return keyDTO(key), nil
}

func (s *Store) DeleteKey(userID, keyID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.keys[keyID]
	if !ok || key.userID != userID {
		return store.ErrNotFound
	}
	delete(s.keys, keyID)
	return nil
}

func (s *Store) ResetKey(userID, keyID int) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.keys[keyID]
	if !ok || key.userID != userID {
		return nil, store.ErrNotFound
	}

	fullKey, err := crypto.GenerateGatewayKey(key.prefix)
	if err != nil {
		return nil, err
	}
	key.keyHash = crypto.HashKey(fullKey)
	return map[string]any{"full_key": fullKey}, nil
}

func (s *Store) userDTO(user *domain.User) map[string]any {
	return map[string]any{
		"id":         user.ID,
		"nickname":   user.Nickname,
		"user_group": user.UserGroup,
		"status":     user.Status,
		"balance":    balanceDTO(user.AvailableBalance, user.FrozenBalance),
	}
}

func balanceDTO(available, frozen string) map[string]any {
	return map[string]any{"available_balance": available, "frozen_balance": frozen}
}

func keyDTO(key *memoryKey) map[string]any {
	return map[string]any{
		"id":           key.id,
		"user_id":      key.userID,
		"key_name":     key.keyName,
		"prefix":       key.prefix,
		"is_active":    key.isActive,
		"last_used_at": key.lastUsedAt,
		"expires_at":   key.expiresAt,
	}
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
// the in-memory store matches the PostgreSQL timestamptz output. Unparseable
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
