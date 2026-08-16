package rule

import (
	"net/http/httptest"
	"testing"
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
