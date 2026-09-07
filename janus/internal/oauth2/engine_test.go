package oauth2

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
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
		ID:               "backend-service",
		ClientName:       "Service A",
		GrantTypes:       []string{"client_credentials"},
		Scopes:           []string{"read:reports", "write:reports"},
		AllowedAudiences: []string{"https://reports.example"},
	}

	resp, err := engine.IssueClientCredentialsToken(client, []string{"read:reports"}, "https://reports.example")
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
	if claims["iss"] != "https://auth.autorix.io" || claims["aud"] != "https://reports.example" {
		t.Errorf("issuer/audience = %v/%v", claims["iss"], claims["aud"])
	}
	if claims["token_use"] != "access_token" {
		t.Errorf("expected access token marker, got %v", claims["token_use"])
	}
	if claims["client_id"] != client.ID {
		t.Errorf("expected client_id ownership claim %q, got %v", client.ID, claims["client_id"])
	}
}

func TestEngine_SeparatesAccessAndIDTokens(t *testing.T) {
	km, err := jwks.NewKeyManager()
	if err != nil {
		t.Fatal(err)
	}
	engine := NewEngine("https://issuer.example", km)

	withOpenID, err := engine.IssueAuthorizationCodeToken(&core.Grant{
		ClientID: "web-client", Subject: "user-1", Scopes: []string{"openid", "profile"},
	})
	if err != nil {
		t.Fatal(err)
	}
	access, err := km.VerifyJWT(withOpenID.AccessToken)
	if err != nil || access["token_use"] != "access_token" {
		t.Fatalf("access token use = %v, err = %v", access["token_use"], err)
	}
	id, err := km.VerifyJWT(withOpenID.IDToken)
	if err != nil || id["token_use"] != "id_token" || id["aud"] != "web-client" {
		t.Fatalf("id token claims = %#v, err = %v", id, err)
	}

	withoutOpenID, err := engine.IssueAuthorizationCodeToken(&core.Grant{
		ClientID: "web-client", Subject: "user-1", Scopes: []string{"profile"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if withoutOpenID.IDToken != "" {
		t.Fatal("ID token issued without openid scope")
	}
}

func TestEngine_AuthorizationCodeResourceChangesOnlyAccessAudience(t *testing.T) {
	km, err := jwks.NewKeyManager()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := NewEngine("https://issuer.example", km).IssueAuthorizationCodeToken(&core.Grant{
		ClientID: "web-client", Subject: "user-1", Scopes: []string{"openid"}, Resource: "https://api.example",
	})
	if err != nil {
		t.Fatal(err)
	}
	access, err := km.VerifyJWT(resp.AccessToken)
	if err != nil || access["iss"] != "https://issuer.example" || access["aud"] != "https://api.example" {
		t.Fatalf("access claims = %#v, err = %v", access, err)
	}
	id, err := km.VerifyJWT(resp.IDToken)
	if err != nil || id["aud"] != "web-client" {
		t.Fatalf("ID token claims = %#v, err = %v", id, err)
	}
}

func TestClientCredentialsAuthorization(t *testing.T) {
	client := &core.OAuth2Client{
		ID:               "backend-service",
		GrantTypes:       []string{"client_credentials"},
		Scopes:           []string{"reports:read", "reports:write"},
		AllowedAudiences: []string{"https://reports.example"},
	}

	for _, tt := range []struct {
		name   string
		client *core.OAuth2Client
		scopes []string
		want   error
	}{
		{name: "registered confidential client scope", client: client, scopes: []string{"reports:read"}},
		{name: "scope escalation", client: client, scopes: []string{"admin"}, want: ErrUnauthorizedScope},
		{name: "grant not registered", client: &core.OAuth2Client{GrantTypes: []string{"authorization_code"}}, want: ErrUnauthorizedGrant},
		{name: "public client", client: &core.OAuth2Client{IsPublic: true, GrantTypes: []string{"client_credentials"}}, want: ErrUnauthorizedGrant},
	} {
		t.Run(tt.name, func(t *testing.T) {
			km, err := jwks.NewKeyManager()
			if err != nil {
				t.Fatal(err)
			}
			_, err = NewEngine("https://issuer.example", km).IssueClientCredentialsToken(tt.client, tt.scopes, "https://reports.example")
			if !errors.Is(err, tt.want) {
				t.Fatalf("IssueClientCredentialsToken() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestRedirectURIAllowed(t *testing.T) {
	client := &core.OAuth2Client{RedirectURIs: []string{"https://app.example/callback", "https://app.example/callback?tenant=one"}}
	for _, tt := range []struct {
		uri  string
		want bool
	}{
		{uri: "https://app.example/callback", want: true},
		{uri: "https://app.example/callback?tenant=one", want: true},
		{uri: "https://app.example/callback/attacker", want: false},
		{uri: "https://app.example/callback?tenant=two", want: false},
		{uri: "https://attacker.example/callback", want: false},
		{uri: "", want: false},
	} {
		t.Run(tt.uri, func(t *testing.T) {
			if got := RedirectURIAllowed(client, tt.uri); got != tt.want {
				t.Fatalf("RedirectURIAllowed(%q) = %t, want %t", tt.uri, got, tt.want)
			}
		})
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
