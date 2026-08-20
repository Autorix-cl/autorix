package credential_test

import (
	"crypto/rand"
	"testing"

	"github.com/autorix/argus/internal/credential"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, credential.KeySize)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generating test key: %v", err)
	}
	return key
}

func TestGenerateSecret_HasPrefixAndMatchingHash(t *testing.T) {
	raw, hash, err := credential.GenerateSecret("aet_")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	if len(raw) < 5 || raw[:4] != "aet_" {
		t.Fatalf("expected prefixed secret, got %q", raw)
	}
	if hash != credential.HashSecret(raw) {
		t.Fatalf("expected hash to match HashSecret(raw)")
	}
	if hash == raw {
		t.Fatalf("hash must not equal the raw secret")
	}
}

func TestGenerateSecret_IsUnpredictable(t *testing.T) {
	a, _, err := credential.GenerateSecret("")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	b, _, err := credential.GenerateSecret("")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	if a == b {
		t.Fatal("expected two generated secrets to differ")
	}
}

func TestConstantTimeEqual(t *testing.T) {
	if !credential.ConstantTimeEqual("abc", "abc") {
		t.Fatal("expected equal strings to compare equal")
	}
	if credential.ConstantTimeEqual("abc", "abd") {
		t.Fatal("expected differing strings to compare unequal")
	}
	if credential.ConstantTimeEqual("", "") {
		t.Fatal("expected two empty strings to never compare equal")
	}
}

func TestEncryptDecrypt_RoundTrips(t *testing.T) {
	key := testKey(t)
	blob, err := credential.Encrypt(key, "super-secret-value")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	got, err := credential.Decrypt(key, blob)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != "super-secret-value" {
		t.Fatalf("expected round-tripped plaintext, got %q", got)
	}
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	key := testKey(t)
	blob, err := credential.Encrypt(key, "super-secret-value")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	wrongKey := testKey(t)
	if _, err := credential.Decrypt(wrongKey, blob); err == nil {
		t.Fatal("expected decryption with the wrong key to fail")
	}
}

func TestDecrypt_TruncatedCiphertextFails(t *testing.T) {
	key := testKey(t)
	if _, err := credential.Decrypt(key, []byte("short")); err == nil {
		t.Fatal("expected decryption of a too-short blob to fail")
	}
}

func TestHeartbeatSignature_DeterministicAndBoundToFields(t *testing.T) {
	sig1 := credential.HeartbeatSignature("secret", "instance-1", 1000, "nonce-1", true, true)
	sig2 := credential.HeartbeatSignature("secret", "instance-1", 1000, "nonce-1", true, true)
	if sig1 != sig2 {
		t.Fatal("expected the same inputs to produce the same signature")
	}

	variants := []string{
		credential.HeartbeatSignature("other-secret", "instance-1", 1000, "nonce-1", true, true),
		credential.HeartbeatSignature("secret", "instance-2", 1000, "nonce-1", true, true),
		credential.HeartbeatSignature("secret", "instance-1", 1001, "nonce-1", true, true),
		credential.HeartbeatSignature("secret", "instance-1", 1000, "nonce-2", true, true),
		credential.HeartbeatSignature("secret", "instance-1", 1000, "nonce-1", false, true),
		credential.HeartbeatSignature("secret", "instance-1", 1000, "nonce-1", true, false),
	}
	for i, v := range variants {
		if v == sig1 {
			t.Fatalf("variant %d unexpectedly matched the base signature", i)
		}
	}
}
