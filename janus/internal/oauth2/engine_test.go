package oauth2

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/autorix/janus/internal/core"
	"github.com/autorix/janus/internal/jwks"
)

func TestVerifyPKCE(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	// 1. Valid S256
	if !VerifyPKCE(verifier, challenge, "S256") {
		t.Errorf("expected PKCE S256 verification to pass")
	}

	// 2. Invalid S256
	if VerifyPKCE("wrong-verifier", challenge, "S256") {
		t.Errorf("expected PKCE S256 verification to fail for wrong verifier")
	}

	// 3. Plain
	if !VerifyPKCE("my-plain-secret", "my-plain-secret", "plain") {
		t.Errorf("expected plain PKCE to pass")
	}
}

func TestEngine_IssueClientCredentialsToken(t *testing.T) {
	km, err := jwks.NewKeyManager()
	if err != nil {
		t.Fatalf("failed to create KeyManager: %v", err)
	}

	engine := NewEngine("https://auth.autorix.io", km)
	client := &core.OAuth2Client{
		ID:         "backend-service",
		ClientName: "Service A",
		Scopes:     []string{"read:reports", "write:reports"},
	}

	resp, err := engine.IssueClientCredentialsToken(client, []string{"read:reports"})
	if err != nil {
		t.Fatalf("IssueClientCredentialsToken failed: %v", err)
	}

	if resp.TokenType != "Bearer" {
		t.Errorf("expected TokenType 'Bearer', got %s", resp.TokenType)
	}

	if len(resp.AccessToken) == 0 {
		t.Fatal("expected non-empty access token")
	}

	// Verify the issued JWT
	claims, err := km.VerifyJWT(resp.AccessToken)
	if err != nil {
		t.Fatalf("failed to verify issued token: %v", err)
	}

	if claims["sub"] != "backend-service" {
		t.Errorf("expected sub 'backend-service', got %v", claims["sub"])
	}
}

func TestAuthenticateClient(t *testing.T) {
	currentHash, _ := HashSecret("current-secret")
	prevHash, _ := HashSecret("prev-secret")
	future := time.Now().Add(1 * time.Hour)
	past := time.Now().Add(-1 * time.Hour)

	// Confidential client with rotation overlap active
	client := &core.OAuth2Client{
		ID:                      "client-1",
		ClientSecretHash:        currentHash,
		PreviousSecretHash:      prevHash,
		PreviousSecretExpiresAt: &future,
		IsPublic:                false,
	}

	// 1. Current secret works
	if !AuthenticateClient(client, "current-secret") {
		t.Error("expected current secret to authenticate")
	}

	// 2. Previous secret works during active overlap
	if !AuthenticateClient(client, "prev-secret") {
		t.Error("expected prev secret to authenticate within overlap window")
	}

	// 3. Wrong secret fails
	if AuthenticateClient(client, "wrong-secret") {
		t.Error("expected wrong secret to fail authentication")
	}

	// 4. Expired overlap window rejects previous secret
	clientExpired := &core.OAuth2Client{
		ID:                      "client-2",
		ClientSecretHash:        currentHash,
		PreviousSecretHash:      prevHash,
		PreviousSecretExpiresAt: &past,
		IsPublic:                false,
	}
	if AuthenticateClient(clientExpired, "prev-secret") {
		t.Error("expected prev secret to fail after overlap expiration")
	}
	if !AuthenticateClient(clientExpired, "current-secret") {
		t.Error("expected current secret to still pass after overlap expiration")
	}

	// 5. Public client always passes
	publicClient := &core.OAuth2Client{ID: "public-client", IsPublic: true}
	if !AuthenticateClient(publicClient, "") {
		t.Error("expected public client to pass")
	}
}

