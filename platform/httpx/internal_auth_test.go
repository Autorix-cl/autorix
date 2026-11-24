package httpx

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	autortls "github.com/autorix/platform/tls"
)

func TestInternalAuth_DisabledWhenUnconfigured(t *testing.T) {
	mw := InternalAuth(InternalAuthConfig{})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/clients", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when unconfigured, got %d", rec.Code)
	}
}

func TestInternalAuth_FailClosedWhenRequiredWithoutSecret(t *testing.T) {
	mw := InternalAuth(InternalAuthConfig{Required: true})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/clients", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when Required=true without credentials, got %d", rec.Code)
	}

	// Health probes must remain accessible even when fail-closed.
	healthRec := httptest.NewRecorder()
	handler.ServeHTTP(healthRec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if healthRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /health/ready bypass, got %d", healthRec.Code)
	}
}

func TestInternalAuth_TokenHeaders(t *testing.T) {
	secret := "super-secret-internal-token-32bytes"
	mw := InternalAuth(InternalAuthConfig{Secret: secret})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	tests := []struct {
		name       string
		headerName string
		headerVal  string
		wantCode   int
	}{
		{"missing token", "", "", http.StatusUnauthorized},
		{"wrong token", InternalTokenHeader, "wrong-token", http.StatusUnauthorized},
		{"canonical internal header", InternalTokenHeader, secret, http.StatusCreated},
		{"console compatibility header", ConsoleTokenHeader, secret, http.StatusCreated},
		{"bearer authorization header", "Authorization", "Bearer " + secret, http.StatusCreated},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/admin/rules", nil)
			if tc.headerName != "" {
				req.Header.Set(tc.headerName, tc.headerVal)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.wantCode {
				t.Fatalf("expected %d, got %d", tc.wantCode, rec.Code)
			}
		})
	}
}

func TestInternalAuth_ProtectPrefixes(t *testing.T) {
	mw := InternalAuth(InternalAuthConfig{
		Secret:          "secret-123",
		ProtectPrefixes: []string{"/admin/", "/scim/"},
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Public route passes without token.
	pubRec := httptest.NewRecorder()
	handler.ServeHTTP(pubRec, httptest.NewRequest(http.MethodGet, "/self-service/login", nil))
	if pubRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on unprotected public route, got %d", pubRec.Code)
	}

	// Admin route without token is rejected.
	adminRec := httptest.NewRecorder()
	handler.ServeHTTP(adminRec, httptest.NewRequest(http.MethodGet, "/admin/identities", nil))
	if adminRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on protected admin route without token, got %d", adminRec.Code)
	}

	// SCIM route with valid token succeeds.
	scimReq := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", nil)
	scimReq.Header.Set(InternalTokenHeader, "secret-123")
	scimRec := httptest.NewRecorder()
	handler.ServeHTTP(scimRec, scimReq)
	if scimRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on protected scim route with valid token, got %d", scimRec.Code)
	}
}

func TestInternalAuth_MTLSPeerCertificate(t *testing.T) {
	spiffeConsole, _ := url.Parse("spiffe://autorix.internal/service/console")
	spiffeOther, _ := url.Parse("spiffe://autorix.internal/service/untrusted")

	mw := InternalAuth(InternalAuthConfig{
		Required:              true,
		AllowedSPIFFEServices: []string{"console"},
	})
	var seenIdentity, seenSpiffe string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenIdentity = autortls.PeerIdentity(r.Context())
		seenSpiffe = autortls.PeerSpiffeID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// Authorized mTLS SPIFFE peer passes even without a shared secret token.
	reqOK := httptest.NewRequest(http.MethodGet, "/admin/clients", nil)
	reqOK.TLS = &tls.ConnectionState{
		VerifiedChains: [][]*x509.Certificate{{
			{
				Subject: pkix.Name{CommonName: "console"},
				URIs:    []*url.URL{spiffeConsole},
			},
		}},
	}
	recOK := httptest.NewRecorder()
	handler.ServeHTTP(recOK, reqOK)
	if recOK.Code != http.StatusOK {
		t.Fatalf("expected 200 for allowed mTLS peer, got %d", recOK.Code)
	}
	if seenIdentity != "console" || seenSpiffe != spiffeConsole.String() {
		t.Fatalf("unexpected peer context: identity=%q spiffe=%q", seenIdentity, seenSpiffe)
	}

	// Unauthorized mTLS SPIFFE peer gets 403 Forbidden.
	reqDeny := httptest.NewRequest(http.MethodGet, "/admin/clients", nil)
	reqDeny.TLS = &tls.ConnectionState{
		VerifiedChains: [][]*x509.Certificate{{
			{
				Subject: pkix.Name{CommonName: "untrusted"},
				URIs:    []*url.URL{spiffeOther},
			},
		}},
	}
	recDeny := httptest.NewRecorder()
	handler.ServeHTTP(recDeny, reqDeny)
	if recDeny.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unauthorized mTLS peer, got %d", recDeny.Code)
	}
}

func TestInternalAuthConfigFromEnv(t *testing.T) {
	t.Setenv("AUTORIX_INTERNAL_AUTH_SECRET", "env-secret")
	t.Setenv("AUTORIX_REQUIRE_INTERNAL_AUTH", "true")
	t.Setenv("AUTORIX_ALLOWED_SPIFFE_SERVICES", "console, aegis")

	cfg := InternalAuthConfigFromEnv("/admin/")
	if cfg.Secret != "env-secret" || !cfg.Required {
		t.Fatalf("unexpected env config: %+v", cfg)
	}
	if len(cfg.AllowedSPIFFEServices) != 2 || cfg.AllowedSPIFFEServices[0] != "console" || cfg.AllowedSPIFFEServices[1] != "aegis" {
		t.Fatalf("unexpected allowed services: %+v", cfg.AllowedSPIFFEServices)
	}
	if len(cfg.ProtectPrefixes) != 1 || cfg.ProtectPrefixes[0] != "/admin/" {
		t.Fatalf("unexpected protect prefixes: %+v", cfg.ProtectPrefixes)
	}
}
