package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const (
	APIKeyPrefix = "op-"
	keyLength    = 32
)

// GenerateAPIKey creates a new API key with the "op-" prefix.
// Returns (rawKey, hash, prefix).
func GenerateAPIKey() (string, string, string, error) {
	bytes := make([]byte, keyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", fmt.Errorf("generate random bytes: %w", err)
	}

	rawKey := APIKeyPrefix + hex.EncodeToString(bytes)
	hash := HashAPIKey(rawKey)
	prefix := rawKey[:11] // "op-" + first 8 hex chars

	return rawKey, hash, prefix, nil
}

// HashAPIKey returns the SHA-256 hash of an API key.
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
