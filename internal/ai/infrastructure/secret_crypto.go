package infrastructure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

// SecretCipher encrypts/decrypts provider API keys at rest with AES-256-GCM.
// Constructed with an explicit key (never a package global — see
// golang-design-patterns: explicit constructors over init()/globals) so
// tests can inject their own and the composition root controls where the
// key comes from (AI_SECRETS_ENCRYPTION_KEY, see cmd/server/main.go).
type SecretCipher struct {
	gcm cipher.AEAD
}

// NewSecretCipher builds a SecretCipher from a 32-byte AES-256 key.
func NewSecretCipher(key []byte) (*SecretCipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("ai: secret cipher key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("ai: build aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("ai: build gcm: %w", err)
	}
	return &SecretCipher{gcm: gcm}, nil
}

// Encrypt returns nonce||ciphertext, ready to store in a BYTEA column.
func (c *SecretCipher) Encrypt(plaintext string) ([]byte, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("ai: generate nonce: %w", err)
	}
	return c.gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// NewSecretCipherFromEnv builds a SecretCipher from a base64-encoded
// 32-byte key in the named environment variable — the one place
// cmd/server/main.go and the ai bootstrap CLIs (cmd/seed-ai-provider,
// cmd/sync-ai-pricing) all read AI_SECRETS_ENCRYPTION_KEY from, so the
// decode-and-validate logic exists once. Generate a key with
// `openssl rand -base64 32`.
func NewSecretCipherFromEnv(envVar string) (*SecretCipher, error) {
	encoded := os.Getenv(envVar)
	if encoded == "" {
		return nil, fmt.Errorf("ai: %s must be set", envVar)
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("ai: %s must be base64: %w", envVar, err)
	}
	return NewSecretCipher(key)
}

// Decrypt reverses Encrypt.
func (c *SecretCipher) Decrypt(ciphertext []byte) (string, error) {
	nonceSize := c.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ai: ciphertext shorter than nonce")
	}
	nonce, sealed := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := c.gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("ai: decrypt: %w", err)
	}
	return string(plaintext), nil
}
