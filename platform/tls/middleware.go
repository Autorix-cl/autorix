package autortls

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"strings"
)

type contextKey string

const (
	PeerIdentityKey contextKey = "autorix.peer_identity"
	PeerSpiffeIDKey contextKey = "autorix.peer_spiffe_id"
)

// PeerIdentity returns the authenticated TLS client certificate CommonName or empty string.
func PeerIdentity(ctx context.Context) string {
	if val, ok := ctx.Value(PeerIdentityKey).(string); ok {
		return val
	}
	return ""
}

// PeerSpiffeID returns the authenticated TLS client certificate SPIFFE URI or empty string.
func PeerSpiffeID(ctx context.Context) string {
	if val, ok := ctx.Value(PeerSpiffeIDKey).(string); ok {
		return val
	}
	return ""
}

// PeerCertificate extracts the verified client certificate from the HTTP request, if present.
func PeerCertificate(r *http.Request) *x509.Certificate {
	if r.TLS != nil && len(r.TLS.VerifiedChains) > 0 && len(r.TLS.VerifiedChains[0]) > 0 {
		return r.TLS.VerifiedChains[0][0]
	}
	return nil
}

// AuthMiddleware extracts peer TLS certificates and injects identity into the request context.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if cert := PeerCertificate(r); cert != nil {
			ctx = context.WithValue(ctx, PeerIdentityKey, cert.Subject.CommonName)
			if len(cert.URIs) > 0 {
				ctx = context.WithValue(ctx, PeerSpiffeIDKey, cert.URIs[0].String())
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePeerIdentity returns a middleware that rejects requests unless the client presented
// a valid mTLS certificate matching one of the allowed services.
func RequirePeerIdentity(allowedServices ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowedServices))
	for _, s := range allowedServices {
		allowed[strings.ToLower(strings.TrimSpace(s))] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cert := PeerCertificate(r)
			if cert == nil {
				writeJSONError(w, http.StatusUnauthorized, "mutual TLS client certificate required")
				return
			}

			peerName := strings.ToLower(cert.Subject.CommonName)
			if len(allowed) > 0 && !allowed[peerName] {
				writeJSONError(w, http.StatusForbidden, "peer identity "+peerName+" is not authorized to call this service")
				return
			}

			ctx := context.WithValue(r.Context(), PeerIdentityKey, cert.Subject.CommonName)
			if len(cert.URIs) > 0 {
				ctx = context.WithValue(ctx, PeerSpiffeIDKey, cert.URIs[0].String())
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
