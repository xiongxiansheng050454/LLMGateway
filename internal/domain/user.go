package domain

import "encoding/json"

type User struct {
	ID               int
	Nickname         string
	UserGroup        string
	Status           string
	AvailableBalance string
	FrozenBalance    string
}

type UserInput struct {
	Nickname  string `json:"nickname"`
	UserGroup string `json:"user_group"`
	Status    string `json:"status"`
	// Password is accepted for API compatibility only. This version does not
	// implement login and never persists a password.
	Password string `json:"password"`
}

type UserStatusInput struct {
	Status string `json:"status"`
}

type RechargeInput struct {
	Amount         string `json:"amount"`
	RelatedOrderID string `json:"related_order_id"`
	Description    string `json:"description"`
}

type BalanceTransaction struct {
	ID           int    `json:"id"`
	TxType       string `json:"tx_type"`
	Amount       string `json:"amount"`
	BalanceAfter string `json:"balance_after"`
	CreatedAt    string `json:"created_at"`
}

type ClientKey struct {
	ID         int     `json:"id"`
	UserID     int     `json:"user_id"`
	KeyName    string  `json:"key_name"`
	Prefix     string  `json:"prefix"`
	IsActive   bool    `json:"is_active"`
	LastUsedAt *string `json:"last_used_at"`
	ExpiresAt  *string `json:"expires_at"`
}

type KeyInput struct {
	KeyName            string          `json:"key_name"`
	Prefix             string          `json:"prefix"`
	Permissions        json.RawMessage `json:"permissions"`
	RateLimitOverrides json.RawMessage `json:"rate_limit_overrides"`
	ExpiresAt          string          `json:"expires_at"`
	IsActive           *bool           `json:"is_active"`
}

type KeyUpdateInput struct {
	IsActive *bool `json:"is_active"`
}
