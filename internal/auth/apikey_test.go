package auth

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey_HasOpPrefix(t *testing.T) {
	rawKey, _, _, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey returned unexpected error: %v", err)
	}

	if !strings.HasPrefix(rawKey, "op-") {
		t.Errorf("expected raw key to start with %q, got %q", "op-", rawKey)
	}
}

func TestGenerateAPIKey_ProducesUniqueKeys(t *testing.T) {
	const iterations = 10
	seen := make(map[string]bool, iterations)

	for i := 0; i < iterations; i++ {
		rawKey, _, _, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("GenerateAPIKey returned unexpected error on iteration %d: %v", i, err)
		}

		if seen[rawKey] {
			t.Fatalf("duplicate key generated on iteration %d: %s", i, rawKey)
		}
		seen[rawKey] = true
	}
}

func TestHashAPIKey_IsDeterministic(t *testing.T) {
	key := "op-abcdef1234567890abcdef1234567890"

	hash1 := HashAPIKey(key)
	hash2 := HashAPIKey(key)

	if hash1 != hash2 {
		t.Errorf("HashAPIKey should be deterministic: got %q and %q", hash1, hash2)
	}
}

func TestGenerateAPIKey_PrefixIsFirst11Chars(t *testing.T) {
	rawKey, _, prefix, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey returned unexpected error: %v", err)
	}

	expected := rawKey[:11]
	if prefix != expected {
		t.Errorf("expected prefix %q (first 11 chars of raw key), got %q", expected, prefix)
	}

	// The prefix should start with "op-" followed by 8 hex characters.
	if !strings.HasPrefix(prefix, "op-") {
		t.Errorf("prefix should start with %q, got %q", "op-", prefix)
	}
	if len(prefix) != 11 {
		t.Errorf("prefix should be 11 characters, got %d", len(prefix))
	}
}

func TestGenerateAPIKey_HashDiffersFromRawKey(t *testing.T) {
	rawKey, hash, _, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey returned unexpected error: %v", err)
	}

	if hash == rawKey {
		t.Error("hash should differ from the raw key")
	}
}

func TestGenerateAPIKey_HashMatchesHashAPIKey(t *testing.T) {
	rawKey, hash, _, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey returned unexpected error: %v", err)
	}

	recomputed := HashAPIKey(rawKey)
	if hash != recomputed {
		t.Errorf("hash from GenerateAPIKey (%q) should match HashAPIKey(%q) result (%q)", hash, rawKey, recomputed)
	}
}

func TestGenerateAPIKey_RawKeyLength(t *testing.T) {
	rawKey, _, _, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey returned unexpected error: %v", err)
	}

	// "op-" (3 chars) + hex-encoded 32 bytes (64 chars) = 67 chars total.
	expectedLen := len(APIKeyPrefix) + 64
	if len(rawKey) != expectedLen {
		t.Errorf("expected raw key length %d, got %d (key: %q)", expectedLen, len(rawKey), rawKey)
	}
}

func TestHashAPIKey_DifferentKeysProduceDifferentHashes(t *testing.T) {
	keys := []string{
		"op-aaaaaaaaaaaaaaaa",
		"op-bbbbbbbbbbbbbbbb",
		"op-cccccccccccccccc",
	}

	hashes := make(map[string]string, len(keys))
	for _, k := range keys {
		h := HashAPIKey(k)
		hashes[k] = h
	}

	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if hashes[keys[i]] == hashes[keys[j]] {
				t.Errorf("keys %q and %q produced the same hash", keys[i], keys[j])
			}
		}
	}
}

func TestHashAPIKey_ReturnsHexString(t *testing.T) {
	hash := HashAPIKey("op-testkey123")

	// SHA-256 produces 32 bytes = 64 hex characters.
	if len(hash) != 64 {
		t.Errorf("expected hash length 64, got %d", len(hash))
	}

	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("hash contains non-hex character: %c", c)
			break
		}
	}
}
