// Package credential centralizes Argus's secret handling (P2-S3-T1/T3/T4):
// high-entropy generation, one-way hashing for direct comparison,
// AES-256-GCM sealing so a raw secret can be recovered server-side for HMAC
// verification without ever storing it in the clear, and constant-time
// comparisons so no code path leaks timing information about a partial
// match. Both the gRPC hot path and the REST admin surface use this package
// instead of each rolling their own.
package credential

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
)

// KeySize is the required length, in bytes, of the AES-256-GCM key passed
// to Encrypt/Decrypt.
const KeySize = 32

// ErrCiphertextTooShort is returned by Decrypt when the blob is too short
// to contain a nonce.
var ErrCiphertextTooShort = errors.New("credential: ciphertext too short")

// GenerateSecret returns a fresh, cryptographically random, hex-encoded
// secret (optionally prefixed, e.g. "aet_" for enrollment tokens) and its
// SHA-256 hash. Only the hash — or, where HMAC verification is needed, an
// Encrypt-sealed copy — should ever be persisted; the raw value is meant to
// be shown to the caller exactly once.
func GenerateSecret(prefix string) (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generating random secret: %w", err)
	}
	raw = prefix + hex.EncodeToString(b)
	return raw, HashSecret(raw), nil
}

// HashSecret returns the hex-encoded SHA-256 digest of raw.
func HashSecret(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ConstantTimeEqual reports whether a and b are equal without leaking
// timing information about how much of the strings matched. Two empty
// strings are never considered equal — an unset stored value must never
// match an unset presented value.
func ConstantTimeEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// Encrypt seals raw under key (which must be KeySize bytes) with
// AES-256-GCM. The returned blob is nonce||ciphertext, so no separate
// nonce storage is required.
func Encrypt(key []byte, raw string) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, []byte(raw), nil), nil
}

// Decrypt reverses Encrypt.
func Decrypt(key []byte, blob []byte) (string, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(blob) < nonceSize {
		return "", ErrCiphertextTooShort
	}
	nonce, ciphertext := blob[:nonceSize], blob[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypting credential: %w", err)
	}
	return string(plaintext), nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("credential: encryption key must be %d bytes, got %d", KeySize, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("building aes cipher: %w", err)
	}
	return cipher.NewGCM(block)
}

// HeartbeatSignature computes the canonical HMAC-SHA256 signature over a
// heartbeat's authenticated fields (P2-S3-T4), hex-encoded. instanceUUID,
// timestampUnix and nonce bind the signature to one specific heartbeat, so
// neither replaying a captured signature under a different message nor
// forging one without the secret succeeds.
func HeartbeatSignature(secret, instanceUUID string, timestampUnix int64, nonce string, live, ready bool) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%s|%d|%s|%t|%t", instanceUUID, timestampUnix, nonce, live, ready)
	return hex.EncodeToString(mac.Sum(nil))
}
