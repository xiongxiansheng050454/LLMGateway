package store

import "LLMGateway/internal/domain"

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
}
