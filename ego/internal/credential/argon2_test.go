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
