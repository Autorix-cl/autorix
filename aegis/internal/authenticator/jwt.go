package authenticator

import (
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

type JWTAuthenticator struct {
	// In production, uses a cached JWKS client from Janus
	publicKeyFunc jwt.Keyfunc
}

func NewJWTAuthenticator(keyFunc jwt.Keyfunc) *JWTAuthenticator {
	return &JWTAuthenticator{publicKeyFunc: keyFunc}
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

	var claims jwt.MapClaims
	if a.publicKeyFunc != nil {
		token, err := jwt.Parse(tokenString, a.publicKeyFunc)
		if err != nil || !token.Valid {
			return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
		}
		claims = token.Claims.(jwt.MapClaims)
	} else {
		// Fallback parser for testing without active network JWKS
		parser := jwt.NewParser()
		token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
		}
		claims = token.Claims.(jwt.MapClaims)
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
