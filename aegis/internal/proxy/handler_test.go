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
		if _, err := fmt.Fprint(w, "OK from Backend"); err != nil {
			t.Errorf("failed writing mock backend response: %v", err)
		}
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

func TestPipelineProxy_PathRewriting(t *testing.T) {
	tests := []struct {
		name         string
		matchURL     string
		stripPrefix  string
		rewrite      string
		requestPath  string
		expectedPath string
	}{
		{
			name:         "Strip prefix basic",
			matchURL:     "/api/v1/<.*>",
			stripPrefix:  "/api/v1",
			requestPath:  "/api/v1/users/42",
			expectedPath: "/users/42",
		},
		{
			name:         "Strip prefix root path",
			matchURL:     "/api/v1",
			stripPrefix:  "/api/v1",
			requestPath:  "/api/v1",
			expectedPath: "/",
		},
		{
			name:         "Rewrite regex replacement",
			matchURL:     "/service/<.*>",
			rewrite:      "/v2/$1",
			requestPath:  "/service/items/99",
			expectedPath: "/v2/items/99",
		},
		{
			name:         "Rewrite multiple groups",
			matchURL:     "/api/<[a-z]+>/<[0-9]+>",
			rewrite:      "/v3/$2/$1",
			requestPath:  "/api/widgets/123",
			expectedPath: "/v3/123/widgets",
		},
		{
			name:         "Strip prefix then proxy to backend with base path",
			matchURL:     "/legacy/<.*>",
			stripPrefix:  "/legacy",
			requestPath:  "/legacy/status",
			expectedPath: "/status",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var receivedPath string
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
			}))
			defer backend.Close()

			ruleYAML := fmt.Sprintf(`
- id: "rewrite-rule"
  match:
    url: "%s"
    methods: ["GET"]
  authenticators:
    - handler: "anonymous"
  authorizer:
    handler: "allow"
  upstream:
    url: "%s"
    strip_prefix: "%s"
    rewrite: "%s"
`, tc.matchURL, backend.URL, tc.stripPrefix, tc.rewrite)

			matcher, err := rule.NewMatcherFromYAML([]byte(ruleYAML))
			if err != nil {
				t.Fatalf("failed to create matcher: %v", err)
			}

			proxyHandler := NewPipelineProxy(
				matcher,
				[]core.Authenticator{&authenticator.AnonymousAuthenticator{}},
				[]core.Authorizer{&authorizer.AllowAuthorizer{}},
				nil,
			)

			req := httptest.NewRequest("GET", tc.requestPath, nil)
			rec := httptest.NewRecorder()

			proxyHandler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
			}

			if receivedPath != tc.expectedPath {
				t.Errorf("expected backend path %q, got %q", tc.expectedPath, receivedPath)
			}
		})
	}
}

func TestPipelineProxy_DryRun(t *testing.T) {
	ruleYAML := `
- id: "dryrun-rule"
  match:
    url: "/api/secure/<.*>"
    methods: ["GET"]
  authenticators:
    - handler: "anonymous"
  authorizer:
    handler: "allow"
  mutators:
    - handler: "header"
      config:
        headers:
          X-Subject: "{{.Subject}}"
  upstream:
    url: "http://backend.internal"
    strip_prefix: "/api/secure"
`
	matcher, err := rule.NewMatcherFromYAML([]byte(ruleYAML))
	if err != nil {
		t.Fatalf("failed to create matcher: %v", err)
	}

	proxyHandler := NewPipelineProxy(
		matcher,
		[]core.Authenticator{&authenticator.AnonymousAuthenticator{}},
		[]core.Authorizer{&authorizer.AllowAuthorizer{}},
		[]core.Mutator{mutator.NewHeaderMutator()},
	)

	req := httptest.NewRequest("GET", "/api/secure/data/1", nil)
	trace, err := proxyHandler.DryRun(req)
	if err != nil {
		t.Fatalf("DryRun unexpected error: %v", err)
	}

	if trace.MatchedRuleID != "dryrun-rule" {
		t.Errorf("expected matched rule dryrun-rule, got %s", trace.MatchedRuleID)
	}
	if trace.FinalVerdict != "allow" {
		t.Errorf("expected verdict allow, got %s", trace.FinalVerdict)
	}
	if len(trace.Steps) != 5 {
		t.Fatalf("expected 5 trace steps, got %d", len(trace.Steps))
	}
	if trace.Steps[0].Stage != "match" || trace.Steps[0].Status != "success" {
		t.Errorf("expected step 0 match success, got %+v", trace.Steps[0])
	}
	if trace.Steps[1].Stage != "authenticator" || trace.Steps[1].Session == nil || trace.Steps[1].Session.Subject != "anonymous" {
		t.Errorf("expected step 1 auth session subject anonymous, got %+v", trace.Steps[1])
	}
	if trace.Steps[2].Stage != "authorizer" || trace.Steps[2].Allowed == nil || !*trace.Steps[2].Allowed {
		t.Errorf("expected step 2 authorizer allowed, got %+v", trace.Steps[2])
	}
	if trace.Steps[3].Stage != "mutator" || trace.Steps[3].MutatedHeaders.Get("X-Subject") != "anonymous" {
		t.Errorf("expected step 3 mutated header X-Subject=anonymous, got %+v", trace.Steps[3])
	}
	if trace.Steps[4].Stage != "upstream" || trace.Steps[4].TargetURL != "http://backend.internal/data/1" {
		t.Errorf("expected step 4 target_url http://backend.internal/data/1, got %s", trace.Steps[4].TargetURL)
	}
}

func TestPipelineProxy_Catalogue(t *testing.T) {
	proxyHandler := NewPipelineProxy(
		nil,
		[]core.Authenticator{&authenticator.AnonymousAuthenticator{}, authenticator.NewJWTAuthenticator(nil), &authenticator.NoopAuthenticator{}},
		[]core.Authorizer{&authorizer.AllowAuthorizer{}, &authorizer.DenyAuthorizer{}, authorizer.NewNexusAuthorizer()},
		[]core.Mutator{mutator.NewHeaderMutator(), &mutator.NoopMutator{}},
	)

	cat := proxyHandler.Catalogue()
	if len(cat.Authenticators) != 3 {
		t.Errorf("expected 3 authenticators in catalogue, got %d", len(cat.Authenticators))
	}
	if len(cat.Authorizers) != 3 {
		t.Errorf("expected 3 authorizers in catalogue, got %d", len(cat.Authorizers))
	}
	if len(cat.Mutators) != 2 {
		t.Errorf("expected 2 mutators in catalogue, got %d", len(cat.Mutators))
	}
}
