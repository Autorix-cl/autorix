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
func (e *Engine) IssueClientCredentialsToken(client *core.OAuth2Client, requestedScopes []string) (*core.TokenResponse, error) {
	lifespan := 1 * time.Hour
	claims := jwks.GenerateClaims(
		e.issuer,
		client.ID,
		e.issuer,
		requestedScopes,
		lifespan,
	)

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

// IssueAuthorizationCodeToken exchanges a valid Grant for an Access Token and ID Token
func (e *Engine) IssueAuthorizationCodeToken(grant *core.Grant) (*core.TokenResponse, error) {
	lifespan := 1 * time.Hour

	// 1. Access Token JWT
	accessClaims := jwks.GenerateClaims(
		e.issuer,
		grant.Subject,
		grant.ClientID,
		grant.Scopes,
		lifespan,
	)
	accessToken, err := e.keyManager.SignJWT(accessClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// 2. ID Token JWT (OpenID Connect)
	idClaims := jwks.GenerateClaims(
		e.issuer,
		grant.Subject,
		grant.ClientID,
		grant.Scopes,
		lifespan,
	)
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
