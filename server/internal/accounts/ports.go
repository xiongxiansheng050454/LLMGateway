package accounts

import (
	"context"
	"encoding/json"
)

// Port covers account reads and single-statement writes that need no
// orchestration. Multistep and rule-bearing operations live on Server and use
// a TxManager.
type Port interface {
	ListUsers(page, pageSize int) (ListResponse[UserDTO], error)
	GetUserBalance(id int) (BalanceDTO, error)
	ListBalanceTransactions(userID, page, pageSize int) (ListResponse[BalanceTransactionDTO], error)

	ListUserKeys(userID, page, pageSize int) (ListResponse[ClientKeyDTO], error)
	ListKeys(page, pageSize int) (ListResponse[ClientKeyDTO], error)

	// AuthenticateKey looks up a gateway key by its hash and returns the raw
	// key + user authentication state. Missing keys return ErrNotFound.
	AuthenticateKey(keyHash string) (*AuthContext, error)
	// UpdateKeyLastUsed records key usage. Missing keys return ErrNotFound.
	UpdateKeyLastUsed(keyID int) error
}

// Tx is the transaction-scoped persistence surface. It exposes only CRUD,
// locking and existence primitives: implementations must not encode business
// rules or multi-step flows, which belong to Server.
type Tx interface {
	GetUser(id int) (User, error)
	InsertUser(nickname, group, status string) (User, error)
	InsertUserBalance(userID int) error
	UpdateUser(id int, nickname, group string) (User, bool, error)
	UpdateUserStatus(id int, status string) (User, bool, error)
	DeleteUser(id int) (bool, error)

	LockUserBalance(userID int) error
	GetUserBalanceText(userID int) (string, error)
	UpdateUserBalance(userID int, available string) (bool, error)
	InsertBalanceTransaction(in BalanceTransactionInput) error
	// GetBalanceTransactionByOrder returns the stored balance_after for an
	// idempotency key, or found=false when no transaction matches.
	GetBalanceTransactionByOrder(userID int, orderID string) (balanceAfter string, found bool, err error)

	GetKey(keyID, userID int) (ClientKey, error)
	InsertKey(in KeyInsert) (int, error)
	UpdateKeyActive(keyID, userID int, active bool) (ClientKey, bool, error)
	UpdateKeySecret(keyID, userID int, keyHash, prefix string) (bool, error)
	DeleteKey(keyID, userID int) (bool, error)

	// DeleteQuotaReservationsForUser and DeleteQuotaReservationsForKey release
	// and remove quota reservations referencing the identity. Identity deletion
	// and quota admission can overlap, so both must run in the caller's
	// transaction to preserve the shared lock order.
	DeleteQuotaReservationsForUser(userID int) error
	DeleteQuotaReservationsForKey(userID, keyID int) error
}

// TxManager runs fn inside a single database transaction.
type TxManager interface {
	InTx(ctx context.Context, fn func(Tx) error) error
}

// BalanceTransactionInput describes a balance ledger entry to persist.
type BalanceTransactionInput struct {
	UserID         int
	TxType         string
	Amount         string
	BalanceAfter   string
	RelatedOrderID string
	Description    string
}

// KeyInsert describes a gateway key row to persist. KeyHash must already be a
// hash of the plaintext key; plaintext is never stored.
type KeyInsert struct {
	UserID             int
	KeyName            string
	Prefix             string
	KeyHash            string
	Permissions        json.RawMessage
	RateLimitOverrides json.RawMessage
	ExpiresAt          string
	IsActive           bool
}
