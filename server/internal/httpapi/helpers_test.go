package httpapi

import "LLMGateway/server/internal/crypto"

const testEncryptionKey = "0123456789abcdef0123456789abcdef"

func testCipher() *crypto.Cipher {
	cipher, err := crypto.NewCipher([]byte(testEncryptionKey))
	if err != nil {
		panic(err)
	}
	return cipher
}
