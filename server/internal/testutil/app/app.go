// Package app wires business module servers over the in-process test store.
// It is test-only; production code must not import it.
package app

import (
	"net/http"

	"LLMGateway/server/internal/accounts"
	"LLMGateway/server/internal/catalog"
	"LLMGateway/server/internal/crypto"
	"LLMGateway/server/internal/quota"
	"LLMGateway/server/internal/ratelimit"
	"LLMGateway/server/internal/testutil/storefake"
)

const testEncryptionKey = "0123456789abcdef0123456789abcdef"

// Deps holds the fake store and the module servers wired over it.
type Deps struct {
	Store     *storefake.Store
	Accounts  *accounts.Server
	Catalog   *catalog.Server
	Quota     *quota.Server
	RateLimit *ratelimit.Server
}

// New builds a Deps over a fresh fake store.
func New() *Deps { return NewWithStore(storefake.New()) }

// NewWithStore wires the module servers over an existing fake store, so tests
// that need an injected clock can reuse it.
func NewWithStore(st *storefake.Store) *Deps {
	cipher, err := crypto.NewCipher([]byte(testEncryptionKey))
	if err != nil {
		panic(err)
	}
	return &Deps{
		Store:    st,
		Accounts: accounts.New(st, st.AccountsTx()),
		Catalog: catalog.New(catalog.Deps{
			Store:  st,
			Health: st,
			Tx:     st.CatalogTx(),
			Cipher: cipher,
			Client: &http.Client{},
		}),
		Quota:     quota.New(st, st.QuotaTx(), nil),
		RateLimit: ratelimit.New(st, nil),
	}
}
