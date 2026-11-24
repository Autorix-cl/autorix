package rule

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/autorix/aegis/internal/core"
)

func TestMatcher(t *testing.T) {
	yamlRules := `
- id: "rule-1"
  match:
    url: "/api/users/<[0-9]+>"
    methods: ["GET", "PUT"]
- id: "rule-2"
  match:
    url: "/public/<.*>"
    methods: ["*"]
`

	matcher, err := NewMatcherFromYAML([]byte(yamlRules))
	if err != nil {
		t.Fatalf("failed to create matcher: %v", err)
	}

	// 1. Match Rule 1
	req1 := httptest.NewRequest("GET", "/api/users/123", nil)
	r1, err := matcher.Match(req1)
	if err != nil {
		t.Fatalf("expected match for req1: %v", err)
	}
	if r1.ID != "rule-1" {
		t.Errorf("expected rule-1, got %s", r1.ID)
	}

	// 2. Mismatch method for Rule 1
	reqDelete := httptest.NewRequest("DELETE", "/api/users/123", nil)
	_, err = matcher.Match(reqDelete)
	if err == nil {
		t.Errorf("expected error for DELETE method on rule-1")
	}

	// 3. Match Wildcard Rule 2
	req2 := httptest.NewRequest("POST", "/public/images/logo.png", nil)
	r2, err := matcher.Match(req2)
	if err != nil {
		t.Fatalf("expected match for req2: %v", err)
	}
	if r2.ID != "rule-2" {
		t.Errorf("expected rule-2, got %s", r2.ID)
	}
}

func TestMatcher_RejectsPathTraversalAndEncodedBypasses(t *testing.T) {
	yamlRules := `
- id: "public"
  match:
    url: "/public/<.*>"
    methods: ["*"]
`
	matcher, err := NewMatcherFromYAML([]byte(yamlRules))
	if err != nil {
		t.Fatalf("failed to create matcher: %v", err)
	}

	malicious := []*http.Request{
		{Method: "GET", URL: &url.URL{Path: "/public/../admin"}},
		{Method: "GET", URL: &url.URL{Path: "/public/foo\x00bar"}},
		{Method: "GET", URL: &url.URL{Path: "/public/foo", RawPath: "/public%2fadmin"}},
		{Method: "GET", URL: &url.URL{Path: "/public/foo", RawPath: "/public%2Fadmin"}},
		{Method: "GET", URL: &url.URL{Path: "/public/foo", RawPath: "/public%5cadmin"}},
		{Method: "GET", URL: &url.URL{Path: "/public/foo", RawPath: "/public/%2e%2e/admin"}},
		{Method: "GET", URL: &url.URL{Path: "/public/foo", RawPath: "/public/%2E/admin"}},
		{Method: "GET", URL: &url.URL{Path: "/public\\admin"}},
	}

	for i, req := range malicious {
		if _, err := matcher.Match(req); err == nil {
			t.Errorf("case %d: expected traversal/bypass path %+v to be rejected", i, req.URL)
		}
	}

	// Clean canonicalization of benign double slashes
	reqClean := httptest.NewRequest("GET", "/public//images/logo.png", nil)
	r, err := matcher.Match(reqClean)
	if err != nil || r.ID != "public" {
		t.Fatalf("expected canonicalized path to match public, got rule=%v err=%v", r, err)
	}
}

func TestValidateRule(t *testing.T) {
	valid := core.Rule{
		ID:       "r1",
		Match:    core.MatchConfig{URL: "/api/<.*>", Methods: []string{"GET"}},
		Upstream: core.UpstreamConfig{URL: "https://backend.internal:8443"},
	}
	if err := ValidateRule(valid); err != nil {
		t.Fatalf("expected valid rule to pass, got %v", err)
	}

	badScheme := valid
	badScheme.Upstream.URL = "file:///etc/passwd"
	if err := ValidateRule(badScheme); err == nil {
		t.Fatal("expected non-http(s) upstream scheme to be rejected")
	}

	badRegex := valid
	badRegex.Match.URL = "/api/[unclosed"
	if err := ValidateRule(badRegex); err == nil {
		t.Fatal("expected invalid regex to be rejected")
	}
}
