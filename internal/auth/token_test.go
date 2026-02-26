package auth

import (
	"testing"
	"time"
)

const testSecret = "test-secret-key-at-least-32-chars!!"

func TestGenerateAccessToken(t *testing.T) {
	token, err := GenerateAccessToken("user-123", "player1", testSecret, 15*time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("GenerateAccessToken() returned empty string")
	}
}

func TestValidateAccessToken_Valid(t *testing.T) {
	token, err := GenerateAccessToken("user-123", "player1", testSecret, 15*time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	claims, err := ValidateAccessToken(token, testSecret)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
	}
	if claims.Username != "player1" {
		t.Errorf("Username = %q, want %q", claims.Username, "player1")
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	token, err := GenerateAccessToken("user-123", "player1", testSecret, -1*time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	_, err = ValidateAccessToken(token, testSecret)
	if err == nil {
		t.Fatal("ValidateAccessToken() expected error for expired token, got nil")
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	token, err := GenerateAccessToken("user-123", "player1", testSecret, 15*time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	_, err = ValidateAccessToken(token, "wrong-secret-key")
	if err == nil {
		t.Fatal("ValidateAccessToken() expected error for wrong secret, got nil")
	}
}

func TestValidateAccessToken_Malformed(t *testing.T) {
	_, err := ValidateAccessToken("not-a-valid-jwt", testSecret)
	if err == nil {
		t.Fatal("ValidateAccessToken() expected error for malformed token, got nil")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	raw, hash, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}
	if raw == "" {
		t.Fatal("GenerateRefreshToken() raw is empty")
	}
	if hash == "" {
		t.Fatal("GenerateRefreshToken() hash is empty")
	}
	if raw == hash {
		t.Fatal("GenerateRefreshToken() raw should not equal hash")
	}
	// hex-encoded 32 bytes = 64 chars
	if len(raw) != 64 {
		t.Errorf("raw length = %d, want 64", len(raw))
	}
	// SHA-256 hash = 64 hex chars
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}
}

func TestHashRefreshToken_Consistent(t *testing.T) {
	raw, hash, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	rehash := HashRefreshToken(raw)
	if rehash != hash {
		t.Errorf("HashRefreshToken() = %q, want %q", rehash, hash)
	}
}

func TestGenerateRefreshToken_Unique(t *testing.T) {
	raw1, _, _ := GenerateRefreshToken()
	raw2, _, _ := GenerateRefreshToken()
	if raw1 == raw2 {
		t.Fatal("GenerateRefreshToken() produced duplicate tokens")
	}
}
