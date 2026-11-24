package mutator

import (
	"net/http/httptest"
	"testing"

	"github.com/autorix/aegis/internal/core"
)

func TestHeaderMutator(t *testing.T) {
	mut := NewHeaderMutator()
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer sensitive-token-123")

	session := &core.Session{
		Subject: "user-42",
		Scopes:  []string{"read", "write"},
	}

	config := map[string]interface{}{
		"headers": map[string]interface{}{
			"X-User-ID": "{{ .Subject }}",
		},
	}

	err := mut.Mutate(req, session, config)
	if err != nil {
		t.Fatalf("Mutate failed: %v", err)
	}

	// 1. Verify injected header
	if req.Header.Get("X-User-ID") != "user-42" {
		t.Errorf("expected X-User-ID 'user-42', got %s", req.Header.Get("X-User-ID"))
	}

	// 2. Verify Authorization header was stripped (Zero Trust)
	if req.Header.Get("Authorization") != "" {
		t.Errorf("expected Authorization header to be deleted")
	}
}

func TestHeaderMutator_DefaultBranchStripsAuthAndEmptyScopes(t *testing.T) {
	mut := NewHeaderMutator()
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer top-secret")
	req.Header.Set("X-User-Scopes", "spoofed:admin")

	session := &core.Session{
		Subject: "user-99",
		Scopes:  nil,
	}

	if err := mut.Mutate(req, session, map[string]interface{}{}); err != nil {
		t.Fatalf("Mutate failed: %v", err)
	}

	if got := req.Header.Get("X-User-ID"); got != "user-99" {
		t.Fatalf("expected X-User-ID 'user-99', got %q", got)
	}
	if got := req.Header.Get("X-User-Scopes"); got != "" {
		t.Fatalf("expected stale/spoofed X-User-Scopes to be deleted when empty, got %q", got)
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("expected Authorization header to be stripped in default branch, got %q", got)
	}
}
