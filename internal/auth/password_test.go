package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword_ReturnsValidBcryptHash(t *testing.T) {
	hash, err := HashPassword("mysecretpassword")
	if err != nil {
		t.Fatalf("HashPassword returned unexpected error: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}

	// A valid bcrypt hash can be parsed by bcrypt.Cost without error.
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("returned hash is not valid bcrypt: %v", err)
	}
	if cost != bcrypt.DefaultCost {
		t.Errorf("expected cost %d, got %d", bcrypt.DefaultCost, cost)
	}
}

func TestCheckPassword_CorrectPassword(t *testing.T) {
	password := "correcthorsebatterystaple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned unexpected error: %v", err)
	}

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword should return true for the correct password")
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	password := "correcthorsebatterystaple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned unexpected error: %v", err)
	}

	wrongPasswords := []string{
		"wrongpassword",
		"correcthorsebatterystapl", // off by one
		"",
		"CORRECTHORSEBATTERYSTAPLE", // case change
	}

	for _, wrong := range wrongPasswords {
		if CheckPassword(wrong, hash) {
			t.Errorf("CheckPassword should return false for wrong password %q", wrong)
		}
	}
}

func TestHashPassword_DifferentPasswordsProduceDifferentHashes(t *testing.T) {
	passwords := []string{
		"password1",
		"password2",
		"password3",
	}

	hashes := make(map[string]string, len(passwords))
	for _, p := range passwords {
		hash, err := HashPassword(p)
		if err != nil {
			t.Fatalf("HashPassword(%q) returned unexpected error: %v", p, err)
		}
		hashes[p] = hash
	}

	// Every pair of passwords should have different hashes.
	for i := 0; i < len(passwords); i++ {
		for j := i + 1; j < len(passwords); j++ {
			if hashes[passwords[i]] == hashes[passwords[j]] {
				t.Errorf("passwords %q and %q produced the same hash", passwords[i], passwords[j])
			}
		}
	}
}

func TestHashPassword_SamePasswordProducesDifferentHashes(t *testing.T) {
	// bcrypt uses a random salt, so hashing the same password twice
	// should produce different hashes.
	hash1, err := HashPassword("samepassword")
	if err != nil {
		t.Fatalf("HashPassword returned unexpected error: %v", err)
	}
	hash2, err := HashPassword("samepassword")
	if err != nil {
		t.Fatalf("HashPassword returned unexpected error: %v", err)
	}

	if hash1 == hash2 {
		t.Error("hashing the same password twice should produce different hashes due to random salt")
	}

	// Both should still verify against the original password.
	if !CheckPassword("samepassword", hash1) {
		t.Error("CheckPassword failed for hash1")
	}
	if !CheckPassword("samepassword", hash2) {
		t.Error("CheckPassword failed for hash2")
	}
}
