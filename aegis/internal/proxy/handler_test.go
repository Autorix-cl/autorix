package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autorix/aegis/internal/authenticator"
	"github.com/autorix/aegis/internal/authorizer"
	"github.com/autorix/aegis/internal/core"
	"github.com/autorix/aegis/internal/mutator"
	"github.com/autorix/aegis/internal/rule"
)

func TestPipelineProxy_EndToEnd(t *testing.T) {
	// 1. Mock upstream backend service
	backendReceivedUserID := ""
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backendReceivedUserID = r.Header.Get("X-User-ID")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK from Backend")
	}))
	defer backend.Close()

	// 2. Define Rule routing to mock backend
	yamlRules := fmt.Sprintf(`
- id: "test-rule"
  match:
    url: "/api/protected"
    methods: ["GET"]
  authenticators:
    - handler: "anonymous"
  authorizer:
    handler: "allow"
  mutators:
    - handler: "header"
      config:
        headers:
          X-User-ID: "anonymous-user"
  upstream:
    url: "%s"
`, backend.URL)

	matcher, err := rule.NewMatcherFromYAML([]byte(yamlRules))
	if err != nil {
		t.Fatalf("failed to create matcher: %v", err)
	}

	pipelineProxy := NewPipelineProxy(
		matcher,
		[]core.Authenticator{&authenticator.AnonymousAuthenticator{}},
		[]core.Authorizer{&authorizer.AllowAuthorizer{}},
		[]core.Mutator{mutator.NewHeaderMutator()},
	)

	// 3. Make request to Aegis
	req := httptest.NewRequest("GET", "/api/protected", nil)
	rec := httptest.NewRecorder()

	pipelineProxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	if backendReceivedUserID != "anonymous-user" {
		t.Errorf("expected backend to receive X-User-ID 'anonymous-user', got %s", backendReceivedUserID)
	}
}
