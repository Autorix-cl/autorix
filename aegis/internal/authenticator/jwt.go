package authenticator

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/autorix/aegis/internal/core"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingToken = errors.New("missing Authorization Bearer token")
	ErrInvalidToken = errors.New("invalid or expired JWT")
)

// JWTAuthenticator verifies access tokens against an operator-provisioned key.
// Trust settings are global, never taken from token headers or rule configuration.
type JWTAuthenticator struct {
	keyFunc  func(context.Context) jwt.Keyfunc
	issuer   string
	audience string
}

func NewJWTAuthenticator(publicKey *rsa.PublicKey, issuer, audience string) (*JWTAuthenticator, error) {
	if publicKey == nil || publicKey.N == nil || publicKey.N.BitLen() < 2048 || publicKey.E < 3 {
		return nil, errors.New("JWT requires an RSA public key of at least 2048 bits")
	}
	if strings.TrimSpace(issuer) == "" || strings.TrimSpace(audience) == "" {
		return nil, errors.New("JWT issuer and audience are required")
	}
	return &JWTAuthenticator{keyFunc: func(context.Context) jwt.Keyfunc {
		return func(*jwt.Token) (interface{}, error) { return publicKey, nil }
	}, issuer: issuer, audience: audience}, nil
}

func (a *JWTAuthenticator) Name() string {
	return "jwt"
}

func (a *JWTAuthenticator) Authenticate(r *http.Request, config map[string]interface{}) (*core.Session, error) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, ErrMissingToken
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// A zero-value authenticator must fail closed, including in test harnesses.
	if a.keyFunc == nil || a.issuer == "" || a.audience == "" {
		return nil, ErrInvalidToken
	}
	if len(config) != 0 {
		return nil, errors.New("JWT per-rule configuration is unsupported; configure trust at startup")
	}
	token, err := jwt.Parse(tokenString, a.keyFunc(r.Context()), jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(a.issuer),
		jwt.WithAudience(a.audience), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	claims := token.Claims.(jwt.MapClaims)
	if tokenUse, _ := claims["token_use"].(string); tokenUse != "access_token" {
		return nil, errors.New("jwt is not an access token")
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, errors.New("jwt missing subject (sub) claim")
	}

	var scopes []string
	if scopeStr, ok := claims["scope"].(string); ok {
		scopes = strings.Fields(scopeStr)
	} else if scopeArr, ok := claims["scopes"].([]interface{}); ok {
		for _, s := range scopeArr {
			if str, ok := s.(string); ok {
				scopes = append(scopes, str)
			}
		}
	}

	return &core.Session{
		Subject: sub,
		Scopes:  scopes,
		Extra:   claims,
		Headers: make(http.Header),
	}, nil
}

// AnonymousAuthenticator provides an empty session for public routes
type AnonymousAuthenticator struct{}

func (a *AnonymousAuthenticator) Name() string { return "anonymous" }
func (a *AnonymousAuthenticator) Authenticate(r *http.Request, config map[string]interface{}) (*core.Session, error) {
	return &core.Session{
		Subject: "anonymous",
		Scopes:  nil,
		Extra:   map[string]interface{}{"is_anonymous": true},
		Headers: make(http.Header),
	}, nil
}

// NoopAuthenticator allows requests to pass through without modifying session
type NoopAuthenticator struct{}

func (a *NoopAuthenticator) Name() string { return "noop" }
func (a *NoopAuthenticator) Authenticate(r *http.Request, config map[string]interface{}) (*core.Session, error) {
	return &core.Session{
		Subject: "noop",
		Extra:   make(map[string]interface{}),
		Headers: make(http.Header),
	}, nil
}
