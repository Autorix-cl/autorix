package mutator

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autorix/aegis/internal/core"
)

func TestHeaderMutator_IdentitySignature(t *testing.T) {
	m := NewHeaderMutator()
	sess := &core.Session{
		Subject: "user_123",
		Scopes:  []string{"read", "write"},
	}

	t.Run("strips spoofed signature when unsigned", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
		req.Header.Set(IdentitySignatureHeader, "v1=spoofed")
		if err := m.Mutate(req, sess, nil); err != nil {
			t.Fatalf("Mutate failed: %v", err)
		}
		if got := req.Header.Get(IdentitySignatureHeader); got != "" {
			t.Fatalf("expected spoofed signature to be stripped, got %q", got)
		}
	})

	t.Run("signs default headers when env secret is set", func(t *testing.T) {
		t.Setenv("AEGIS_IDENTITY_SIGNING_SECRET", "shared-proxy-secret")
		req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
		if err := m.Mutate(req, sess, nil); err != nil {
			t.Fatalf("Mutate failed: %v", err)
		}
		want := SignIdentityHeaders("user_123", "", "", "read write", "shared-proxy-secret")
		if got := req.Header.Get(IdentitySignatureHeader); got != want {
			t.Fatalf("expected signature %q, got %q", want, got)
		}
	})

	t.Run("signs templated headers with config signing_secret", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
		cfg := map[string]interface{}{
			"signing_secret": "rule-secret",
			"headers": map[string]interface{}{
				"X-User-ID":    "{{.Subject}}",
				"X-User-Email": "alice@autorix.io",
				"X-User-Roles": "admin,operator",
			},
		}
		if err := m.Mutate(req, sess, cfg); err != nil {
			t.Fatalf("Mutate failed: %v", err)
		}
		want := SignIdentityHeaders("user_123", "alice@autorix.io", "admin,operator", "", "rule-secret")
		if got := req.Header.Get(IdentitySignatureHeader); got != want {
			t.Fatalf("expected signature %q, got %q", want, got)
		}
	})
}
