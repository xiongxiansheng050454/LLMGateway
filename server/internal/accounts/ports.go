package accounts

import (
	"encoding/json"
)

// CanonicalJSON is a serialization-consistency helper (not a business rule):
// it re-encodes JSON into a compact, key-sorted form so test fakes (raw input)
// and PostgreSQL (JSONB) return byte-identical values. Empty input
// returns nil.
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
type Port interface {
	ListUsers(page, pageSize int) (ListResponse[UserDTO], error)
	CreateUser(UserInput) (UserDTO, error)
	UpdateUser(int, UserInput) (UserDTO, error)
	UpdateUserStatus(int, string) (UserDTO, error)
	DeleteUser(int) error
	RechargeUser(int, RechargeInput) (BalanceUpdateDTO, error)
	GetUserBalance(int) (BalanceDTO, error)
	ListBalanceTransactions(userID, page, pageSize int) (ListResponse[BalanceTransactionDTO], error)

	ListUserKeys(userID, page, pageSize int) (ListResponse[ClientKeyDTO], error)
	ListKeys(page, pageSize int) (ListResponse[ClientKeyDTO], error)
	CreateKey(userID int, in KeyInput) (KeySecretDTO, error)
	UpdateKey(userID, keyID int, in KeyUpdateInput) (ClientKeyDTO, error)
	DeleteKey(userID, keyID int) error
	ResetKey(userID, keyID int) (KeySecretDTO, error)

	// AuthenticateKey looks up a gateway key by its hash and returns the raw
	// key + user authentication state. Missing keys return ErrNotFound.
	AuthenticateKey(keyHash string) (*AuthContext, error)
	// UpdateKeyLastUsed records key usage. Missing keys return ErrNotFound.
	UpdateKeyLastUsed(keyID int) error
	// DebitUserBalance deducts amount (6 decimals) from the available balance
	// inside a transaction and records a consume transaction. Insufficient
	// balance returns ErrInvalid; a missing user returns ErrNotFound.
	DebitUserBalance(userID int, amount string, description string) (BalanceUpdateDTO, error)
}
