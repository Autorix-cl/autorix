package authenticator

import (
	"context"
	"crypto/rsa"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// NewJWKSAuthenticator trusts only the operator-configured JWKS endpoint.
// The context owns background refresh; callers must cancel it on shutdown.
func NewJWKSAuthenticator(ctx context.Context, endpoint, issuer, audience string, allowInsecureHTTP bool) (*JWTAuthenticator, error) {
	if strings.TrimSpace(issuer) == "" || strings.TrimSpace(audience) == "" {
		return nil, errors.New("JWT issuer and audience are required")
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "https" && !(allowInsecureHTTP && u.Scheme == "http")) {
		return nil, errors.New("JWT_JWKS_URL requires an absolute HTTPS URL without credentials or fragment; insecure HTTP is development-only")
	}
	failFirstFetch := false
	keys, err := keyfunc.NewDefaultOverrideCtx(ctx, []string{endpoint}, keyfunc.Override{
		Client: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("JWKS redirects are not allowed")
		}},
		HTTPTimeout:               5 * time.Second,
		NoErrorReturnFirstHTTPReq: &failFirstFetch,
		RefreshInterval:           time.Minute,
		RateLimitWaitMax:          100 * time.Millisecond,
	})
	if err != nil {
		return nil, err
	}
	return &JWTAuthenticator{issuer: issuer, audience: audience, keyFunc: func(ctx context.Context) jwt.Keyfunc {
		return func(token *jwt.Token) (interface{}, error) {
			// Requiring kid prevents fallback to an unconstrained verification key set.
			if kid, ok := token.Header["kid"].(string); !ok || kid == "" {
				return nil, ErrInvalidToken
			}
			key, err := keys.KeyfuncCtx(ctx)(token)
			if err != nil {
				return nil, err
			}
			rsaKey, ok := key.(*rsa.PublicKey)
			if !ok || rsaKey.N == nil || rsaKey.N.BitLen() < 2048 {
				return nil, ErrInvalidToken
			}
			return rsaKey, nil
		}
	}}, nil
}
