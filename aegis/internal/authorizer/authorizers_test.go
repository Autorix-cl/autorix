package authorizer

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autorix/aegis/internal/core"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestNexusAuthorizer_FailsClosedOnInvalidSubjectsAndMissingConfig(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/doc/1", nil)
	cfg := map[string]interface{}{
		"namespace": "document",
		"relation":  "viewer",
		"object":    "doc-1",
	}

	// Unconfigured endpoint fails closed even for valid subject.
	unconfigured := NewNexusAuthorizer()
	allowed, err := unconfigured.Authorize(req, &core.Session{Subject: "alice"}, cfg)
	if allowed || err == nil {
		t.Fatalf("expected unconfigured NexusAuthorizer to fail closed, got allowed=%v err=%v", allowed, err)
	}

	// Synthetic / anonymous / empty / nil subjects are always rejected.
	configured := NewNexusAuthorizerWithEndpoint("http://nexus.internal", &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			t.Fatalf("nexus endpoint should not be called for invalid subject")
			return nil, nil
		}),
	})
	for _, sess := range []*core.Session{nil, {Subject: ""}, {Subject: "anonymous"}, {Subject: "noop"}, {Subject: "NOOP"}} {
		allowed, err := configured.Authorize(req, sess, cfg)
		if allowed || err == nil {
			t.Fatalf("expected subject %+v to be denied, got allowed=%v err=%v", sess, allowed, err)
		}
	}

	// Missing rule config fails closed.
	allowed, err = configured.Authorize(req, &core.Session{Subject: "alice"}, map[string]interface{}{"namespace": "document"})
	if allowed || err == nil {
		t.Fatalf("expected missing config to fail closed, got allowed=%v err=%v", allowed, err)
	}
}

func TestNexusAuthorizer_EvaluatesRemoteCheckAndFailsClosedOnError(t *testing.T) {
	cfg := map[string]interface{}{
		"namespace": "document",
		"relation":  "editor",
		"object":    "doc-42",
	}
	req := httptest.NewRequest(http.MethodPost, "/api/doc/42", nil)
	sess := &core.Session{Subject: "alice"}

	// Explicit allow
	var captured map[string]interface{}
	allowClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
				t.Fatalf("decode check request: %v", err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"allowed":true}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	authzAllow := NewNexusAuthorizerWithEndpoint("http://nexus.internal", allowClient)
	allowed, err := authzAllow.Authorize(req, sess, cfg)
	if !allowed || err != nil {
		t.Fatalf("expected allowed=true, got allowed=%v err=%v", allowed, err)
	}
	if captured["subject_id"] != "alice" || captured["object"] != "doc-42" || captured["relation"] != "editor" {
		t.Fatalf("unexpected nexus check payload: %+v", captured)
	}

	// Explicit deny
	denyClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"allowed":false,"reason":"missing relation"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	allowed, err = NewNexusAuthorizerWithEndpoint("http://nexus.internal", denyClient).Authorize(req, sess, cfg)
	if allowed || err == nil {
		t.Fatalf("expected explicit deny to fail closed, got allowed=%v err=%v", allowed, err)
	}

	// Network failure -> fail closed
	errClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		}),
	}
	allowed, err = NewNexusAuthorizerWithEndpoint("http://nexus.internal", errClient).Authorize(req, sess, cfg)
	if allowed || err == nil {
		t.Fatalf("expected network error to fail closed, got allowed=%v err=%v", allowed, err)
	}
}
