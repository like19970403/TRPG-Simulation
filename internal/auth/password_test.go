package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("testpass123", bcrypt.MinCost)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword() returned empty string")
	}
	if hash == "testpass123" {
		t.Fatal("HashPassword() returned plaintext password")
	}
}

func TestCheckPassword_Valid(t *testing.T) {
	hash, err := HashPassword("testpass123", bcrypt.MinCost)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if err := CheckPassword("testpass123", hash); err != nil {
		t.Errorf("CheckPassword() valid password error = %v", err)
	}
}

func TestCheckPassword_Invalid(t *testing.T) {
	hash, err := HashPassword("testpass123", bcrypt.MinCost)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if err := CheckPassword("wrongpassword", hash); err == nil {
		t.Error("CheckPassword() expected error for wrong password, got nil")
	}
}

func TestHashPassword_EmptyPassword(t *testing.T) {
	hash, err := HashPassword("", bcrypt.MinCost)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	// bcrypt accepts empty string; verification should still work
	if err := CheckPassword("", hash); err != nil {
		t.Errorf("CheckPassword() empty password error = %v", err)
	}
}

func TestHashPassword_MaxLength(t *testing.T) {
	// bcrypt truncates at 72 bytes
	pass72 := string(make([]byte, 72))
	for i := range pass72 {
		pass72 = pass72[:i] + "a" + pass72[i+1:]
	}

	hash, err := HashPassword(pass72, bcrypt.MinCost)
	if err != nil {
		t.Fatalf("HashPassword() 72-byte password error = %v", err)
	}
	if err := CheckPassword(pass72, hash); err != nil {
		t.Errorf("CheckPassword() 72-byte password error = %v", err)
	}
}
