package credential

import (
	"testing"
)

func TestArgon2_HashAndVerify(t *testing.T) {
	password := "SecretMasterKey#2026"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if !valid {
		t.Error("expected valid password verification")
	}

	invalid, err := VerifyPassword("WrongPassword123", hash)
	if err != nil {
		t.Fatalf("VerifyPassword with wrong password failed: %v", err)
	}
	if invalid {
		t.Error("expected invalid password to fail verification")
	}
}

