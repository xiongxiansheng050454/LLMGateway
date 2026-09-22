// Package app wires business module servers over the in-process test store.
// It is test-only; production code must not import it.
package app

import (
	"LLMGateway/server/internal/accounts"
	"LLMGateway/server/internal/testutil/storefake"
)

// Deps holds the fake store and the module servers wired over it.
type Deps struct {
	Store    *storefake.Store
	Accounts *accounts.Server
}

// New builds a Deps over a fresh fake store.
func New() *Deps { return NewWithStore(storefake.New()) }

// NewWithStore wires the module servers over an existing fake store, so tests
// that need an injected clock can reuse it.
func NewWithStore(st *storefake.Store) *Deps {
	return &Deps{
		Store:    st,
		Accounts: accounts.New(st, st.AccountsTx()),
	}
}
