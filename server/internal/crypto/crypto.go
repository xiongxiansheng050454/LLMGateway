// Package crypto holds the secret-handling primitives: symmetric encryption for
// upstream channel api keys and hashing for gateway API keys.
//
// It is a leaf package: it must not depend on internal/store or internal/httpapi.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ErrInvalidKey is returned when an encryption key is missing or malformed.
var ErrInvalidKey = errors.New("invalid encryption key")

// defaultGatewayKeyPrefix matches the gateway key format used by the dashboard.
const defaultGatewayKeyPrefix = "sk-"

// Cipher encrypts and decrypts upstream channel api keys with AES-GCM.
type Cipher struct {
	aead cipher.AEAD
}

// NewCipher builds a Cipher from a raw key of 16, 24 or 32 bytes.
func NewCipher(key []byte) (*Cipher, error) {
	switch len(key) {
	case 16, 24, 32:
	default:
		return nil, fmt.Errorf("%w: length must be 16, 24 or 32 bytes", ErrInvalidKey)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	return &Cipher{aead: aead}, nil
}

// NewCipherFromEnv builds a Cipher from the named environment variable. A
// missing or malformed value is an error; callers must not fall back to
// plaintext storage.
func NewCipherFromEnv(name string) (*Cipher, error) {
	value := os.Getenv(name)
	if value == "" {
		return nil, fmt.Errorf("%w: environment variable %s is not set", ErrInvalidKey, name)
	}
	return NewCipher([]byte(value))
}

// Encrypt returns a base64 string containing the random nonce and ciphertext.
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. A wrong key or tampered input returns an error.
func (c *Cipher) Decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	nonceSize := c.aead.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("crypto: ciphertext too short")
	}

	plaintext, err := c.aead.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// GenerateGatewayKey returns a new random gateway key. Only the hash should be
// persisted; the plaintext is returned to the caller once.
func GenerateGatewayKey(prefix string) (string, error) {
	if prefix == "" {
		prefix = defaultGatewayKeyPrefix
	}

	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf), nil
}

// HashKey returns the SHA-256 hex digest of a gateway key. Surrounding
// whitespace is trimmed so equivalent inputs hash identically.
func HashKey(key string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(key)))
	return hex.EncodeToString(sum[:])
}

// EqualHashes compares two hex digests in constant time.
func EqualHashes(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
