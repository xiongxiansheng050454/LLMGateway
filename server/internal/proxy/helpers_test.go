package proxy

import (
	"net/http"
	"time"

	"LLMGateway/server/internal/catalog"
	"LLMGateway/server/internal/crypto"
	"LLMGateway/server/internal/quota"
	"LLMGateway/server/internal/ratelimit"
	"LLMGateway/server/internal/testutil/storefake"
)

const testEncryptionKey = "0123456789abcdef0123456789abcdef"

// newTestCatalog wires a catalog server over a fake store for proxy tests.
func newTestCatalog(st *storefake.Store) *catalog.Server {
	cipher, err := crypto.NewCipher([]byte(testEncryptionKey))
	if err != nil {
		panic(err)
	}
	return catalog.New(catalog.Deps{
		Store:  st,
		Health: st,
		Tx:     st.CatalogTx(),
		Cipher: cipher,
		Client: &http.Client{},
	})
}

func newTestQuota(st *storefake.Store, now func() time.Time) *quota.Server {
	if now == nil {
		now = time.Now
	}
	return quota.New(st, st.QuotaTx(), now)
}

func newTestRateLimit(st *storefake.Store, now func() time.Time) *ratelimit.Server {
	if now == nil {
		now = time.Now
	}
	return ratelimit.New(st, now)
}
