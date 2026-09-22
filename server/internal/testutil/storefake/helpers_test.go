package storefake

import (
	"net/http"

	"LLMGateway/server/internal/catalog"
	"LLMGateway/server/internal/crypto"
)

// testEncryptionKey is a fixed 32-byte key used only by tests.
const testEncryptionKey = "0123456789abcdef0123456789abcdef"

func testCipher() *crypto.Cipher {
	cipher, err := crypto.NewCipher([]byte(testEncryptionKey))
	if err != nil {
		panic(err)
	}
	return cipher
}

// newCatalog wires a catalog server over the fake store for tests that exercise
// catalog rules.
func newCatalog(st *Store) *catalog.Server {
	return catalog.New(catalog.Deps{
		Store:  st,
		Health: st,
		Tx:     st.CatalogTx(),
		Cipher: testCipher(),
		Client: &http.Client{},
	})
}
