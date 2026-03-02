package auth

import (
	"testing"
	"time"
)

func TestGenerate_ProducesValidTokenString(t *testing.T) {
	svc := NewJWTService("test-secret-key", 1)

	token, err := svc.Generate("user-123", "user@example.com")
	if err != nil {
		t.Fatalf("Generate returned unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("Generate returned empty token")
	}
}

func TestValidate_SucceedsWithValidToken(t *testing.T) {
	svc := NewJWTService("test-secret-key", 1)

	token, err := svc.Generate("user-123", "user@example.com")
	if err != nil {
		t.Fatalf("Generate returned unexpected error: %v", err)
	}

	claims, err := svc.Validate(token)
	if err != nil {
		t.Fatalf("Validate returned unexpected error: %v", err)
	}
	if claims == nil {
		t.Fatal("Validate returned nil claims")
	}
}

func TestValidate_FailsWithInvalidToken(t *testing.T) {
	svc := NewJWTService("test-secret-key", 1)

	tests := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"garbage", "not-a-jwt-token"},
		{"tampered token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiZmFrZSJ9.invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := svc.Validate(tt.token)
			if err == nil {
				t.Error("Validate should return an error for an invalid token")
			}
			if claims != nil {
				t.Error("Validate should return nil claims for an invalid token")
			}
		})
	}
}

func TestValidate_FailsWithWrongSecret(t *testing.T) {
	svc1 := NewJWTService("secret-one", 1)
	svc2 := NewJWTService("secret-two", 1)

	token, err := svc1.Generate("user-123", "user@example.com")
	if err != nil {
		t.Fatalf("Generate returned unexpected error: %v", err)
	}

	claims, err := svc2.Validate(token)
	if err == nil {
		t.Error("Validate should fail when using a different secret")
	}
	if claims != nil {
		t.Error("Validate should return nil claims when using a different secret")
	}
}

func TestValidate_FailsWithExpiredToken(t *testing.T) {
	// Create a service with an expiration in the past by using a negative duration hack.
	// Since NewJWTService takes hours as int, we instead create the service manually
	// with a very short expiration to test expiry.
	svc := &JWTService{
		secret:     []byte("test-secret-key"),
		expiration: 1 * time.Millisecond,
	}

	token, err := svc.Generate("user-123", "user@example.com")
	if err != nil {
		t.Fatalf("Generate returned unexpected error: %v", err)
	}

	// Wait for the token to expire.
	time.Sleep(10 * time.Millisecond)

	claims, err := svc.Validate(token)
	if err == nil {
		t.Error("Validate should return an error for an expired token")
	}
	if claims != nil {
		t.Error("Validate should return nil claims for an expired token")
	}
}

func TestValidate_ClaimsContainCorrectUserIDAndEmail(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		email  string
	}{
		{"basic user", "user-123", "user@example.com"},
		{"uuid user", "550e8400-e29b-41d4-a716-446655440000", "admin@openpath.io"},
		{"empty fields", "", ""},
	}

	svc := NewJWTService("test-secret-key", 1)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := svc.Generate(tt.userID, tt.email)
			if err != nil {
				t.Fatalf("Generate returned unexpected error: %v", err)
			}

			claims, err := svc.Validate(token)
			if err != nil {
				t.Fatalf("Validate returned unexpected error: %v", err)
			}

			if claims.UserID != tt.userID {
				t.Errorf("expected UserID %q, got %q", tt.userID, claims.UserID)
			}
			if claims.Email != tt.email {
				t.Errorf("expected Email %q, got %q", tt.email, claims.Email)
			}
		})
	}
}

func TestValidate_ClaimsContainCorrectIssuer(t *testing.T) {
	svc := NewJWTService("test-secret-key", 1)

	token, err := svc.Generate("user-123", "user@example.com")
	if err != nil {
		t.Fatalf("Generate returned unexpected error: %v", err)
	}

	claims, err := svc.Validate(token)
	if err != nil {
		t.Fatalf("Validate returned unexpected error: %v", err)
	}

	if claims.Issuer != "openpaths" {
		t.Errorf("expected issuer %q, got %q", "openpaths", claims.Issuer)
	}
}

func TestGenerate_DifferentUsersProduceDifferentTokens(t *testing.T) {
	svc := NewJWTService("test-secret-key", 1)

	token1, err := svc.Generate("user-123", "user1@example.com")
	if err != nil {
		t.Fatalf("Generate returned unexpected error: %v", err)
	}

	token2, err := svc.Generate("user-456", "user2@example.com")
	if err != nil {
		t.Fatalf("Generate returned unexpected error: %v", err)
	}

	if token1 == token2 {
		t.Error("tokens for different users should be different")
	}
}
