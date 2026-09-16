package store

import (
	"encoding/json"

	"LLMGateway/server/internal/domain"
)

// CanonicalJSON re-encodes JSON into a compact, key-sorted form so the memory
// store (raw input) and PostgreSQL (JSONB) return byte-identical values.
// Empty input returns nil.
func CanonicalJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return value
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return value
	}
	return encoded
}

// UserStore covers downstream users, their balances, balance transactions and
// gateway API keys.
//
// Implementations must never return a gateway key's plaintext after creation or
// reset, and must never persist the plaintext key (only its hash).
type UserStore interface {
	ListUsers(page, pageSize int) (domain.ListResponse, error)
	CreateUser(domain.UserInput) (map[string]any, error)
	UpdateUser(int, domain.UserInput) (map[string]any, error)
	UpdateUserStatus(int, string) (map[string]any, error)
	DeleteUser(int) error
	RechargeUser(int, domain.RechargeInput) (map[string]any, error)
	GetUserBalance(int) (map[string]any, error)
	ListBalanceTransactions(userID, page, pageSize int) (domain.ListResponse, error)

	ListUserKeys(userID, page, pageSize int) (domain.ListResponse, error)
	ListKeys(page, pageSize int) (domain.ListResponse, error)
	CreateKey(userID int, in domain.KeyInput) (map[string]any, error)
	UpdateKey(userID, keyID int, in domain.KeyUpdateInput) (map[string]any, error)
	DeleteKey(userID, keyID int) error
	ResetKey(userID, keyID int) (map[string]any, error)

	// AuthenticateKey looks up a gateway key by its hash and returns the raw
	// key + user authentication state. Missing keys return ErrNotFound.
	AuthenticateKey(keyHash string) (*domain.AuthContext, error)
	// UpdateKeyLastUsed records key usage. Missing keys return ErrNotFound.
	UpdateKeyLastUsed(keyID int) error
	// DebitUserBalance deducts amount (6 decimals) from the available balance
	// inside a transaction and records a consume transaction. Insufficient
	// balance returns ErrInvalid; a missing user returns ErrNotFound.
	DebitUserBalance(userID int, amount string, description string) (map[string]any, error)
}
