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
