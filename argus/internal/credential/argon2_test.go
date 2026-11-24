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

func TestArgon2_RejectsInvalidAndOutOfRangeParams(t *testing.T) {
	validSalt := "AAAAAAAAAAAAAAAAAAAAAA"                     // 16 bytes
	validKey := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" // 32 bytes

	cases := []struct {
		name string
		hash string
	}{
		{"zero_memory", "$argon2id$v=19$m=0,t=3,p=4$" + validSalt + "$" + validKey},
		{"zero_iterations", "$argon2id$v=19$m=65536,t=0,p=4$" + validSalt + "$" + validKey},
		{"zero_parallelism", "$argon2id$v=19$m=65536,t=3,p=0$" + validSalt + "$" + validKey},
		{"excessive_memory", "$argon2id$v=19$m=4194304,t=3,p=4$" + validSalt + "$" + validKey},
		{"excessive_iterations", "$argon2id$v=19$m=65536,t=999,p=4$" + validSalt + "$" + validKey},
		{"excessive_parallelism", "$argon2id$v=19$m=65536,t=3,p=64$" + validSalt + "$" + validKey},
		{"missing_param", "$argon2id$v=19$m=65536,t=3$" + validSalt + "$" + validKey},
		{"duplicate_param", "$argon2id$v=19$m=65536,m=65536,t=3,p=4$" + validSalt + "$" + validKey},
		{"unknown_param", "$argon2id$v=19$m=65536,t=3,p=4,x=1$" + validSalt + "$" + validKey},
		{"short_salt", "$argon2id$v=19$m=65536,t=3,p=4$AAAA$" + validSalt},
		{"short_key", "$argon2id$v=19$m=65536,t=3,p=4$" + validSalt + "$AAAA"},
		{"leading_prefix_junk", "junk$argon2id$v=19$m=65536,t=3,p=4$" + validSalt + "$" + validKey},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, err := VerifyPassword("password", tc.hash)
			if err == nil || ok {
				t.Fatalf("expected error and ok=false for %s, got ok=%v err=%v", tc.name, ok, err)
			}
		})
	}
}
