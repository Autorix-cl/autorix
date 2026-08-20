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

func TestKeyManager_RotateKey(t *testing.T) {
	km, err := NewKeyManager()
	if err != nil {
		t.Fatalf("failed to create KeyManager: %v", err)
	}

	oldKid := km.KeyID()
	oldClaims := GenerateClaims("https://auth.autorix.io", "user-1", "client-1", []string{"openid"}, 1*time.Hour)
	oldToken, err := km.SignJWT(oldClaims)
	if err != nil {
		t.Fatalf("SignJWT failed with initial key: %v", err)
	}

	// Rotate key
	newJWK, err := km.RotateKey()
	if err != nil {
		t.Fatalf("RotateKey failed: %v", err)
	}

	newKid := km.KeyID()
	if newKid == oldKid {
		t.Errorf("expected new kid after rotation, but stayed %s", oldKid)
	}
	if newJWK.Kid != newKid {
		t.Errorf("RotateKey returned JWK kid %s, want %s", newJWK.Kid, newKid)
	}

	// Verify old token signed before rotation still verifies successfully (rollover verification)
	parsedOld, err := km.VerifyJWT(oldToken)
	if err != nil {
		t.Fatalf("VerifyJWT failed on old token after rotation: %v", err)
	}
	if parsedOld["sub"] != "user-1" {
		t.Errorf("expected sub 'user-1', got %v", parsedOld["sub"])
	}

	// Sign a new token with rotated key
	newClaims := GenerateClaims("https://auth.autorix.io", "user-2", "client-2", []string{"openid"}, 1*time.Hour)
	newToken, err := km.SignJWT(newClaims)
	if err != nil {
		t.Fatalf("SignJWT failed with rotated key: %v", err)
	}

	parsedNew, err := km.VerifyJWT(newToken)
	if err != nil {
		t.Fatalf("VerifyJWT failed on new token: %v", err)
	}
	if parsedNew["sub"] != "user-2" {
		t.Errorf("expected sub 'user-2', got %v", parsedNew["sub"])
	}

	// Export JWKS should contain both keys
	jwksResp := km.ExportJWKS()
	if len(jwksResp.Keys) != 2 {
		t.Fatalf("expected 2 keys in JWKS after rotation, got %d", len(jwksResp.Keys))
	}
	// First key should be current signing key
	if jwksResp.Keys[0].Kid != newKid {
		t.Errorf("expected first key to be %s, got %s", newKid, jwksResp.Keys[0].Kid)
	}
	if jwksResp.Keys[1].Kid != oldKid {
		t.Errorf("expected second key to be %s, got %s", oldKid, jwksResp.Keys[1].Kid)
	}
}

