package oauth2

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/autorix/janus/internal/core"
	"github.com/autorix/janus/internal/jwks"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidClient        = errors.New("invalid client credentials")
	ErrInvalidGrant         = errors.New("invalid or expired authorization grant")
	ErrInvalidPKCEVerifier  = errors.New("pkce code_verifier does not match code_challenge")
	ErrUnsupportedGrantType = errors.New("unsupported grant_type")
	ErrUnauthorizedGrant    = errors.New("client is not authorized for this grant type")
	ErrUnauthorizedScope    = errors.New("requested scope is not authorized for this client")
	ErrUnauthorizedAudience = errors.New("requested resource audience is not authorized for this client")
)

type Engine struct {
	issuer     string
	keyManager *jwks.KeyManager
}

func NewEngine(issuer string, km *jwks.KeyManager) *Engine {
	return &Engine{
		issuer:     issuer,
		keyManager: km,
	}
}

// VerifyPKCE validates the code_verifier against the stored code_challenge using SHA-256 (RFC 7636)
func VerifyPKCE(verifier, challenge, method string) bool {
	if method == "plain" {
		return verifier == challenge
	}

	// Default & recommended: S256
	h := sha256.Sum256([]byte(verifier))
	calculated := base64.RawURLEncoding.EncodeToString(h[:])
	return calculated == challenge
}

// IssueClientCredentialsToken issues an Access Token JWT for Machine-to-Machine clients
func (e *Engine) IssueClientCredentialsToken(client *core.OAuth2Client, requestedScopes []string, audience string) (*core.TokenResponse, error) {
	if client == nil || client.IsPublic || !SupportsGrantType(client, "client_credentials") {
		return nil, ErrUnauthorizedGrant
	}
	if !ScopesAllowed(client, requestedScopes) {
		return nil, ErrUnauthorizedScope
	}
	if !AudienceAllowed(client, audience) {
		return nil, ErrUnauthorizedAudience
	}
	lifespan := 1 * time.Hour
	claims := jwks.GenerateClaims(
		e.issuer,
		client.ID,
		audience,
		requestedScopes,
		lifespan,
	)
	claims["token_use"] = "access_token"
	// RFC 8707 resource audiences are not necessarily the OAuth client ID.
	// Preserve the token owner explicitly so introspection and revocation do
	// not confuse an API resource identifier with the confidential client.
	claims["client_id"] = client.ID

	signedJWT, err := e.keyManager.SignJWT(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	return &core.TokenResponse{
		AccessToken: signedJWT,
		TokenType:   "Bearer",
		ExpiresIn:   int64(lifespan.Seconds()),
		Scope:       strings.Join(requestedScopes, " "),
	}, nil
}

// AudienceAllowed requires an exact registered RFC 8707 resource value. It
// deliberately does not fall back to the issuer when no resource is supplied.
func AudienceAllowed(client *core.OAuth2Client, audience string) bool {
	if client == nil || audience == "" {
		return false
	}
	for _, allowed := range client.AllowedAudiences {
		if audience == allowed {
			return true
		}
	}
	return false
}

// SupportsGrantType reports whether a client was explicitly registered for a
// grant. Token endpoints must not infer authorization from successful client
// authentication alone.
func SupportsGrantType(client *core.OAuth2Client, grantType string) bool {
	if client == nil {
		return false
	}
	for _, allowed := range client.GrantTypes {
		if allowed == grantType {
			return true
		}
	}
	return false
}

// ScopesAllowed requires every requested scope to be present in the client's
// registered scope allowlist. Empty scope requests are valid only when the
// caller intentionally requests no permissions.
func ScopesAllowed(client *core.OAuth2Client, requested []string) bool {
	if client == nil {
		return false
	}
	allowed := make(map[string]struct{}, len(client.Scopes))
	for _, scope := range client.Scopes {
		allowed[scope] = struct{}{}
	}
	for _, scope := range requested {
		if _, ok := allowed[scope]; !ok {
			return false
		}
	}
	return true
}

// RedirectURIAllowed performs exact matching. Prefix or host-only comparisons
// let an attacker register a nearby URI and receive authorization codes.
func RedirectURIAllowed(client *core.OAuth2Client, redirectURI string) bool {
	if client == nil || redirectURI == "" {
		return false
	}
	for _, allowed := range client.RedirectURIs {
		if allowed == redirectURI {
			return true
		}
	}
	return false
}

// IssueAuthorizationCodeToken exchanges a valid Grant for an Access Token and ID Token
func (e *Engine) IssueAuthorizationCodeToken(grant *core.Grant) (*core.TokenResponse, error) {
	lifespan := 1 * time.Hour
	accessAudience := grant.ClientID
	if grant.Resource != "" {
		accessAudience = grant.Resource
	}

	// 1. Access Token JWT
	accessClaims := jwks.GenerateClaims(
		e.issuer,
		grant.Subject,
		accessAudience,
		grant.Scopes,
		lifespan,
	)
	accessClaims["token_use"] = "access_token"
	accessToken, err := e.keyManager.SignJWT(accessClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// ID tokens are an OpenID Connect artifact and must not be issued unless
	// the client was granted the openid scope.
	if !hasScope(grant.Scopes, "openid") {
		return &core.TokenResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   int64(lifespan.Seconds()),
			Scope:       strings.Join(grant.Scopes, " "),
		}, nil
	}

	// 2. ID Token JWT (OpenID Connect). Its audience remains the client, but
	// resource servers must reject it based on token_use.
	idClaims := jwks.GenerateClaims(
		e.issuer,
		grant.Subject,
		grant.ClientID,
		grant.Scopes,
		lifespan,
	)
	idClaims["token_use"] = "id_token"
	idClaims["auth_time"] = time.Now().Unix()
	idToken, err := e.keyManager.SignJWT(idClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to sign id_token: %w", err)
	}

	return &core.TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(lifespan.Seconds()),
		IDToken:     idToken,
		Scope:       strings.Join(grant.Scopes, " "),
	}, nil
}

func hasScope(scopes []string, wanted string) bool {
	for _, scope := range scopes {
		if scope == wanted {
			return true
		}
	}
	return false
}

// HashSecret helper for client secret hashing
func HashSecret(secret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifySecret helper for client secret verification
func VerifySecret(secret, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)) == nil
}

// AuthenticateClient checks credentials for public and confidential clients,
// including previous secret validity during the rotation overlap window.
func AuthenticateClient(client *core.OAuth2Client, secret string) bool {
	if client.IsPublic {
		return true
	}
	if client.ClientSecretHash != "" && VerifySecret(secret, client.ClientSecretHash) {
		return true
	}
	if client.PreviousSecretHash != "" && client.PreviousSecretExpiresAt != nil &&
		time.Now().Before(*client.PreviousSecretExpiresAt) &&
		VerifySecret(secret, client.PreviousSecretHash) {
		return true
	}
	return false
}
