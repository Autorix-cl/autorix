package jwks

import (
	"testing"
	"time"
)

func TestKeyManager_SignAndVerify(t *testing.T) {
	km, err := NewKeyManager()
	if err != nil {
		t.Fatalf("failed to create KeyManager: %v", err)
	}

	claims := GenerateClaims("https://auth.autorix.io", "user-123", "client-abc", []string{"openid", "profile"}, 1*time.Hour)

	// 1. Sign JWT
	tokenString, err := km.SignJWT(claims)
	if err != nil {
		t.Fatalf("SignJWT failed: %v", err)
	}

	if len(tokenString) == 0 {
		t.Fatal("expected non-empty token string")
	}

	// 2. Verify JWT
	parsedClaims, err := km.VerifyJWT(tokenString)
	if err != nil {
		t.Fatalf("VerifyJWT failed: %v", err)
	}

	if parsedClaims["sub"] != "user-123" {
		t.Errorf("expected sub 'user-123', got %v", parsedClaims["sub"])
	}

	if parsedClaims["iss"] != "https://auth.autorix.io" {
		t.Errorf("expected iss 'https://auth.autorix.io', got %v", parsedClaims["iss"])
	}

	// 3. Verify JWKS Export
	jwks := km.ExportJWKS()
	if len(jwks.Keys) != 1 {
		t.Fatalf("expected 1 key in JWKS, got %d", len(jwks.Keys))
	}

	if jwks.Keys[0].Kid != km.KeyID() {
		t.Errorf("expected kid %s, got %s", km.KeyID(), jwks.Keys[0].Kid)
	}

	if jwks.Keys[0].Alg != "RS256" || jwks.Keys[0].Kty != "RSA" {
		t.Errorf("expected RS256/RSA, got %s/%s", jwks.Keys[0].Alg, jwks.Keys[0].Kty)
	}
}
