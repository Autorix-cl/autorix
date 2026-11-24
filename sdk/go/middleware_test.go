package autorix

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware_SignedIdentityVerification(t *testing.T) {
	client := NewClient(Config{})
	secret := "test-identity-signing-secret"

	echoUser := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("X-Resolved-User", u.ID)
		w.WriteHeader(http.StatusOK)
	})

	t.Run("accepts valid signature", func(t *testing.T) {
		h := client.MiddlewareWithConfig(MiddlewareConfig{SigningSecret: secret})(echoUser)
		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		req.Header.Set("X-User-ID", "usr_42")
		req.Header.Set("X-User-Email", "alice@autorix.io")
		req.Header.Set("X-User-Roles", "admin, viewer")
		req.Header.Set("X-User-Scopes", "read write")
		req.Header.Set(IdentitySignatureHeader, SignIdentityHeaders("usr_42", "alice@autorix.io", "admin, viewer", "read write", secret))

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if rec.Header().Get("X-Resolved-User") != "usr_42" {
			t.Fatalf("expected usr_42, got %q", rec.Header().Get("X-Resolved-User"))
		}
	})

	t.Run("rejects tampered header", func(t *testing.T) {
		h := client.MiddlewareWithConfig(MiddlewareConfig{SigningSecret: secret})(echoUser)
		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		req.Header.Set("X-User-ID", "usr_admin")
		req.Header.Set(IdentitySignatureHeader, SignIdentityHeaders("usr_42", "", "", "", secret))

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for tampered X-User-ID, got %d", rec.Code)
		}
	})

	t.Run("rejects missing signature when secret configured", func(t *testing.T) {
		h := client.MiddlewareWithConfig(MiddlewareConfig{SigningSecret: secret})(echoUser)
		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		req.Header.Set("X-User-ID", "usr_spoofed")

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for unsigned identity header, got %d", rec.Code)
		}
	})

	t.Run("fail-closed when RequireSignature is true without secret", func(t *testing.T) {
		h := client.MiddlewareWithConfig(MiddlewareConfig{RequireSignature: true})(echoUser)
		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		req.Header.Set("X-User-ID", "usr_spoofed")

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 when RequireSignature=true and secret empty, got %d", rec.Code)
		}
	})

	t.Run("env secret activates verification on default Middleware", func(t *testing.T) {
		t.Setenv("AUTORIX_IDENTITY_SIGNING_SECRET", secret)
		h := client.Middleware(echoUser)
		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		req.Header.Set("X-User-ID", "usr_spoofed")

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 when env secret is set and signature missing, got %d", rec.Code)
		}
	})
}
