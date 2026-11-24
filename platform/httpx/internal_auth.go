package httpx

import (
	"context"
	"crypto/subtle"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	autortls "github.com/autorix/platform/tls"
)

const (
	// InternalTokenHeader is the canonical HTTP header carrying an internal
	// service-to-service shared authentication token.
	InternalTokenHeader = "X-Autorix-Internal-Token" //nolint:gosec // header name, not a credential
	// ConsoleTokenHeader is the legacy Console BFF header accepted alongside
	// InternalTokenHeader for backward compatibility.
	ConsoleTokenHeader = "X-Autorix-Console-Token" //nolint:gosec // header name, not a credential
)

var defaultInternalAuthBypassPaths = []string{
	"/health",
	"/health/alive",
	"/health/ready",
	"/info",
	"/metrics",
}

// InternalAuthConfig controls how InternalAuth authenticates service-to-service
// and Console BFF calls on private/admin HTTP listeners.
type InternalAuthConfig struct {
	// Secret is the shared token accepted via X-Autorix-Internal-Token,
	// X-Autorix-Console-Token, or Authorization: Bearer <token>.
	Secret string
	// Required forces fail-closed authentication even if Secret is empty
	// (requiring a verified mTLS peer certificate when no token secret is set).
	Required bool
	// AllowedSPIFFEServices restricts which mTLS SPIFFE services (or CommonNames
	// when no SPIFFE URI SAN is present) are authorized. Empty allows any verified
	// client certificate issued by the configured CA.
	AllowedSPIFFEServices []string
	// BypassPaths lists exact paths (or "/" suffixed prefixes) that bypass internal
	// authentication (such as liveness/readiness probes and Prometheus scrapes).
	// When empty, default health/info/metrics paths are used.
	BypassPaths []string
	// ProtectPrefixes, when non-empty, scopes enforcement exclusively to requests
	// whose URL path matches one of the entries (prefix match if ending in "/",
	// exact or subpath match otherwise). When empty, every non-bypass path is protected.
	ProtectPrefixes []string
}

// InternalAuthConfigFromEnv builds an InternalAuthConfig from standard Autorix
// environment variables. Setting AUTORIX_INTERNAL_AUTH_SECRET or
// AUTORIX_CONSOLE_INTERNAL_TOKEN enables token authentication; setting
// AUTORIX_REQUIRE_INTERNAL_AUTH=true forces fail-closed mode.
func InternalAuthConfigFromEnv(protectPrefixes ...string) InternalAuthConfig {
	secret := strings.TrimSpace(os.Getenv("AUTORIX_INTERNAL_AUTH_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("AUTORIX_CONSOLE_INTERNAL_TOKEN"))
	}
	reqEnv := strings.TrimSpace(os.Getenv("AUTORIX_REQUIRE_INTERNAL_AUTH"))
	required := strings.EqualFold(reqEnv, "true") || reqEnv == "1"

	var allowed []string
	if raw := strings.TrimSpace(os.Getenv("AUTORIX_ALLOWED_SPIFFE_SERVICES")); raw != "" {
		for _, item := range strings.Split(raw, ",") {
			if svc := strings.ToLower(strings.TrimSpace(item)); svc != "" {
				allowed = append(allowed, svc)
			}
		}
	}

	return InternalAuthConfig{
		Secret:                secret,
		Required:              required,
		AllowedSPIFFEServices: allowed,
		ProtectPrefixes:       protectPrefixes,
	}
}

// InternalAuth enforces service-to-service authentication via either:
//  1. A verified mTLS client certificate (matching AllowedSPIFFEServices if set), OR
//  2. A constant-time verified shared secret token in X-Autorix-Internal-Token,
//     X-Autorix-Console-Token, or Authorization: Bearer <token>.
//
// When neither Secret nor Required is configured, the middleware is a no-op so
// local unit tests without internal secrets continue to work unchanged.
func InternalAuth(cfg InternalAuthConfig) Middleware {
	secretBytes := []byte(strings.TrimSpace(cfg.Secret))
	if !cfg.Required && len(secretBytes) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	bypass := cfg.BypassPaths
	if len(bypass) == 0 {
		bypass = defaultInternalAuthBypassPaths
	}

	allowed := make(map[string]bool, len(cfg.AllowedSPIFFEServices))
	for _, s := range cfg.AllowedSPIFFEServices {
		if normalized := strings.ToLower(strings.TrimSpace(s)); normalized != "" {
			allowed[normalized] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if matchesAnyPath(r.URL.Path, bypass) {
				next.ServeHTTP(w, r)
				return
			}
			if len(cfg.ProtectPrefixes) > 0 && !matchesAnyPath(r.URL.Path, cfg.ProtectPrefixes) {
				next.ServeHTTP(w, r)
				return
			}

			// 1. Check verified mTLS client certificate first.
			if cert := autortls.PeerCertificate(r); cert != nil {
				authorized, spiffeID := authorizePeerCert(cert, allowed)
				if !authorized {
					writeInternalAuthError(w, http.StatusForbidden, "peer mTLS identity is not authorized for this internal endpoint")
					return
				}
				ctx := context.WithValue(r.Context(), autortls.PeerIdentityKey, cert.Subject.CommonName)
				if spiffeID != "" {
					ctx = context.WithValue(ctx, autortls.PeerSpiffeIDKey, spiffeID)
				}
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// 2. Fall back to constant-time shared internal token check.
			if len(secretBytes) > 0 && hasValidInternalToken(r, secretBytes) {
				next.ServeHTTP(w, r)
				return
			}

			writeInternalAuthError(w, http.StatusUnauthorized, "internal service authentication required")
		})
	}
}

func matchesAnyPath(reqPath string, patterns []string) bool {
	for _, p := range patterns {
		if p == "" {
			continue
		}
		if strings.HasSuffix(p, "/") {
			if strings.HasPrefix(reqPath, p) || reqPath == strings.TrimSuffix(p, "/") {
				return true
			}
			continue
		}
		if reqPath == p || strings.HasPrefix(reqPath, p+"/") {
			return true
		}
	}
	return false
}

func hasValidInternalToken(r *http.Request, expected []byte) bool {
	candidates := []string{
		r.Header.Get(InternalTokenHeader),
		r.Header.Get(ConsoleTokenHeader),
	}
	if auth := r.Header.Get("Authorization"); len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		candidates = append(candidates, strings.TrimSpace(auth[7:]))
	}
	for _, c := range candidates {
		if c != "" && subtle.ConstantTimeCompare([]byte(c), expected) == 1 {
			return true
		}
	}
	return false
}

func authorizePeerCert(cert *x509.Certificate, allowed map[string]bool) (bool, string) {
	hasSpiffeURI := false
	var firstSpiffeID string
	for _, u := range cert.URIs {
		if u == nil || !strings.EqualFold(u.Scheme, "spiffe") {
			continue
		}
		hasSpiffeURI = true
		uriStr := u.String()
		if firstSpiffeID == "" {
			firstSpiffeID = uriStr
		}
		if svc, ok := autortls.ExtractServiceFromSPIFFE(uriStr); ok {
			if len(allowed) == 0 || allowed[svc] {
				return true, uriStr
			}
		}
	}
	if !hasSpiffeURI {
		peerName := strings.ToLower(cert.Subject.CommonName)
		if len(allowed) == 0 || allowed[peerName] {
			return true, ""
		}
	}
	return false, firstSpiffeID
}

func writeInternalAuthError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
