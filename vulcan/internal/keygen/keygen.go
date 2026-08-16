package keygen

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type GeneratedKey struct {
	Prefix   string
	RawToken string
	Hint     string
	KeyHash  string
	RootKey  string
}

// GenerateAPIKey creates a cryptographically secure API key with prefix and hash
func GenerateAPIKey(isLive bool) (*GeneratedKey, error) {
	prefix := "av_live"
	if !isLive {
		prefix = "av_test"
	}

	// 1. Generate 32 bytes of secure random entropy for the secret token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	secretHex := hex.EncodeToString(tokenBytes)
	rawToken := fmt.Sprintf("%s_%s", prefix, secretHex)

	// 2. Compute key hint (last 4 characters)
	hint := secretHex[len(secretHex)-4:]

	// 3. Compute deterministic SHA-256 hash for database indexing
	keyHash := HashKey(rawToken)

	// 4. Generate 32 bytes for the Root Signature Key (used for Macaroons)
	rootBytes := make([]byte, 32)
	if _, err := rand.Read(rootBytes); err != nil {
		return nil, fmt.Errorf("failed to generate root signature key: %w", err)
	}
	rootKey := hex.EncodeToString(rootBytes)

	return &GeneratedKey{
		Prefix:   prefix,
		RawToken: rawToken,
		Hint:     hint,
		KeyHash:  keyHash,
		RootKey:  rootKey,
	}, nil
}

// HashKey computes SHA-256 hash of a raw token string
func HashKey(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
