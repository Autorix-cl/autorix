package autorix

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// IdentitySignatureHeader carries the HMAC-SHA256 signature computed by Aegis
// over X-User-ID, X-User-Email, X-User-Roles, and X-User-Scopes.
const IdentitySignatureHeader = "X-Autorix-Identity-Signature"

// MiddlewareConfig controls how identity headers injected by Aegis are verified.
type MiddlewareConfig struct {
	// SigningSecret is the shared HMAC-SHA256 key configured on Aegis's header mutator.
	// When non-empty, requests carrying X-User-ID must include a valid
	// X-Autorix-Identity-Signature header or they are rejected with 401 Unauthorized.
	SigningSecret string
	// RequireSignature enforces fail-closed verification even if SigningSecret is empty
	// (rejecting all identity headers until a secret is configured).
	RequireSignature bool
}

// SignIdentityHeaders computes the canonical v1 HMAC-SHA256 signature for
// Aegis-injected identity headers.
func SignIdentityHeaders(userID, email, roles, scopes, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "v1\n%s\n%s\n%s\n%s", userID, email, roles, scopes)
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifyIdentityHeaders validates X-Autorix-Identity-Signature against the
// request's X-User-* headers using constant-time comparison.
func VerifyIdentityHeaders(h http.Header, secret string) bool {
	if strings.TrimSpace(secret) == "" {
		return false
	}
	got := strings.TrimSpace(h.Get(IdentitySignatureHeader))
	if got == "" {
		return false
	}
	expected := SignIdentityHeaders(
		h.Get("X-User-ID"),
		h.Get("X-User-Email"),
		h.Get("X-User-Roles"),
		h.Get("X-User-Scopes"),
		secret,
	)
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

func defaultIdentitySigningSecret() string {
	if s := strings.TrimSpace(os.Getenv("AUTORIX_IDENTITY_SIGNING_SECRET")); s != "" {
		return s
	}
	return strings.TrimSpace(os.Getenv("AEGIS_IDENTITY_SIGNING_SECRET"))
}

// Middleware extracts identity headers injected by Autorix Aegis (X-User-ID, X-User-Email, X-User-Roles).
// When AUTORIX_IDENTITY_SIGNING_SECRET or AEGIS_IDENTITY_SIGNING_SECRET is set in the environment,
// it cryptographically verifies X-Autorix-Identity-Signature before trusting the headers.
func (c *Client) Middleware(next http.Handler) http.Handler {
	return c.MiddlewareWithConfig(MiddlewareConfig{
		SigningSecret: defaultIdentitySigningSecret(),
	})(next)
}

// MiddlewareWithConfig returns an HTTP middleware that extracts and optionally
// verifies Aegis-signed identity headers.
func (c *Client) MiddlewareWithConfig(cfg MiddlewareConfig) func(http.Handler) http.Handler {
	secret := strings.TrimSpace(cfg.SigningSecret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get("X-User-ID")
			if userID != "" {
				if cfg.RequireSignature || secret != "" {
					if !VerifyIdentityHeaders(r.Header, secret) {
						http.Error(w, `{"error":"unauthorized","message":"Invalid or missing identity header signature"}`, http.StatusUnauthorized)
						return
					}
				}

				email := r.Header.Get("X-User-Email")
				rawRoles := r.Header.Get("X-User-Roles")

				var roles []string
				if rawRoles != "" {
					for _, role := range strings.Split(rawRoles, ",") {
						if trimmed := strings.TrimSpace(role); trimmed != "" {
							roles = append(roles, trimmed)
						}
					}
				}

				user := &User{
					ID:        userID,
					Email:     email,
					Roles:     roles,
					IsMachine: strings.HasPrefix(userID, "service_") || strings.HasPrefix(userID, "av_"),
				}

				ctx := WithUser(r.Context(), user)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth enforces that a request must have an authenticated identity.
func (c *Client) RequireAuth(next http.Handler) http.Handler {
	return c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := UserFromContext(r.Context())
		if !ok {
			http.Error(w, `{"error":"unauthorized","message":"Missing authenticated identity"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// RequirePermission enforces a Zanzibar ReBAC permission check before executing the handler.
func (c *Client) RequirePermission(namespace, relation string, getObjectID func(r *http.Request) string, next http.Handler) http.Handler {
	return c.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := UserFromContext(r.Context())
		objectID := getObjectID(r)

		allowed, err := c.Nexus.Check(r.Context(), CheckRequest{
			Namespace: namespace,
			Object:    objectID,
			Relation:  relation,
			SubjectID: user.ID,
		})

		if err != nil || !allowed {
			http.Error(w, `{"error":"forbidden","message":"Access denied by policy"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}))
}
