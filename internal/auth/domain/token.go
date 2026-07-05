package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// GenerateSecureToken returns a high-entropy random token (raw, sent to the
// user over email) and its SHA-256 hash (stored in the DB). Only the hash is
// ever persisted, so a leaked DB row cannot be replayed as a working token.
func GenerateSecureToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashToken(raw), nil
}

// HashToken hashes a raw token the same way GenerateSecureToken does, so a
// token received back from the client can be looked up by its hash.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
