package credential

import (
	"testing"
)

func TestArgon2Hasher(t *testing.T) {
	hasher := NewHasher(&Argon2Params{
		Memory:      16 * 1024,
		Iterations:  1,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	})

	password := "SecureP@ssw0rd!2026"

	// 1. Generate Hash
	hash, err := hasher.GenerateHash(password)
	if err != nil {
		t.Fatalf("GenerateHash failed: %v", err)
	}

	if len(hash) == 0 {
		t.Fatal("expected non-empty hash")
	}

	// 2. Compare valid password
	match, err := hasher.ComparePasswordAndHash(password, hash)
	if err != nil {
		t.Fatalf("ComparePasswordAndHash error: %v", err)
	}
	if !match {
		t.Errorf("expected password to match hash")
	}

	// 3. Compare invalid password
	match, err = hasher.ComparePasswordAndHash("WrongPassword!", hash)
	if err != nil {
		t.Fatalf("ComparePasswordAndHash error: %v", err)
	}
	if match {
		t.Errorf("expected wrong password not to match hash")
	}
}

func TestGenerateSecureRandomString(t *testing.T) {
	s1, err := GenerateSecureRandomString(16)
	if err != nil {
		t.Fatalf("GenerateSecureRandomString: %v", err)
	}
	if len(s1) != 16 {
		t.Errorf("expected length 16, got %d", len(s1))
	}

	s2, err := GenerateSecureRandomString(16)
	if err != nil {
		t.Fatalf("GenerateSecureRandomString: %v", err)
	}
	if s1 == s2 {
		t.Errorf("expected distinct random strings")
	}
}

func TestGenerateRecoveryToken(t *testing.T) {
	raw, hash, err := GenerateRecoveryToken()
	if err != nil {
		t.Fatalf("GenerateRecoveryToken: %v", err)
	}
	if len(raw) == 0 || len(hash) == 0 {
		t.Fatalf("expected non-empty raw token and hash")
	}
	if raw == hash {
		t.Errorf("raw token and hash must differ")
	}
}

func TestGenerateTOTPSecret_And_ValidateCode(t *testing.T) {
	secret, qrURI, err := GenerateTOTPSecret("Autorix", "alice@example.com")
	if err != nil {
		t.Fatalf("GenerateTOTPSecret: %v", err)
	}
	if len(secret) == 0 {
		t.Fatal("expected non-empty TOTP secret")
	}
	if len(qrURI) == 0 {
		t.Fatal("expected non-empty TOTP QR URI")
	}

	// Generate expected code for current time
	code, err := GenerateTOTPCode(secret, 0)
	if err != nil {
		t.Fatalf("GenerateTOTPCode: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6-digit code, got %s", code)
	}

	// Validate valid code
	valid := ValidateTOTPCode(secret, code)
	if !valid {
		t.Errorf("expected TOTP code %s to be valid for secret %s", code, secret)
	}

	// Validate invalid code
	invalid := ValidateTOTPCode(secret, "000000")
	if code != "000000" && invalid {
		t.Errorf("expected invalid TOTP code to fail validation")
	}
}

func TestGenerateBackupCodes(t *testing.T) {
	plainCodes, hashedCodes, err := GenerateBackupCodes(8)
	if err != nil {
		t.Fatalf("GenerateBackupCodes: %v", err)
	}
	if len(plainCodes) != 8 || len(hashedCodes) != 8 {
		t.Fatalf("expected 8 plain and 8 hashed backup codes")
	}
	for i := 0; i < 8; i++ {
		if plainCodes[i] == hashedCodes[i] {
			t.Errorf("plain code and hash at index %d should differ", i)
		}
		if len(plainCodes[i]) == 0 || len(hashedCodes[i]) == 0 {
			t.Errorf("empty code at index %d", i)
		}
	}
}

