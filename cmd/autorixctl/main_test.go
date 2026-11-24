package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func mockHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestDoRequest_InjectsHeadersAndExecutes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Errorf("expected bearer auth, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := Config{
		ArgusURL: server.URL,
		Token:    "test-secret",
	}

	resp, err := doRequest(cfg, "GET", server.URL+"/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestDoRequest_WithCustomClient(t *testing.T) {
	var gotAuth, gotCT, gotURL string
	cfg := Config{
		ArgusURL: "https://argus.internal",
		Token:    "test-secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				gotAuth = r.Header.Get("Authorization")
				gotCT = r.Header.Get("Content-Type")
				gotURL = r.URL.String()
				return mockHTTPResponse(http.StatusOK, `{"status":"ok"}`), nil
			}),
		},
	}

	resp, err := doRequest(cfg, http.MethodGet, "https://argus.internal/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if gotAuth != "Bearer test-secret" {
		t.Errorf("expected bearer auth, got %q", gotAuth)
	}
	if gotCT != "application/json" {
		t.Errorf("expected application/json, got %q", gotCT)
	}
	if gotURL != "https://argus.internal/test" {
		t.Errorf("unexpected URL %q", gotURL)
	}
}

func TestGetEnv_Fallback(t *testing.T) {
	_ = os.Unsetenv("TEST_AUTORIX_VAR")
	val := getEnv("TEST_AUTORIX_VAR", "default_val")
	if val != "default_val" {
		t.Errorf("expected default_val, got %s", val)
	}

	t.Setenv("TEST_AUTORIX_VAR", "custom")
	val = getEnv("TEST_AUTORIX_VAR", "default_val")
	if val != "custom" {
		t.Errorf("expected custom, got %s", val)
	}
}

func TestRunFleet_EscapesEnvQueryAndChecksStatus(t *testing.T) {
	var gotRawQuery string
	cfg := Config{
		ArgusURL: "https://argus.internal/",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				gotRawQuery = r.URL.RawQuery
				return mockHTTPResponse(http.StatusOK, `{"instances":[]}`), nil
			}),
		},
	}

	var out bytes.Buffer
	if err := runFleet(cfg, []string{"list", "--env", "prod&admin=true#frag"}, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotRawQuery != "env=prod%26admin%3Dtrue%23frag" {
		t.Fatalf("expected properly escaped env query, got %q", gotRawQuery)
	}

	cfgErr := Config{
		ArgusURL: "https://argus.internal",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return mockHTTPResponse(http.StatusInternalServerError, `{"error":"boom"}`), nil
			}),
		},
	}
	if err := runFleet(cfgErr, []string{"list"}, &out); err == nil {
		t.Fatal("expected error on HTTP 500 from fleet endpoint")
	}
}

func TestRunCheck_ValidatesStatusAndJSON(t *testing.T) {
	t.Run("allow response", func(t *testing.T) {
		var gotPayload map[string]interface{}
		cfg := Config{
			NexusURL: "https://nexus.internal",
			HTTPClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
						t.Errorf("decode payload: %v", err)
					}
					return mockHTTPResponse(http.StatusOK, `{"allowed":true}`), nil
				}),
			},
		}
		var out bytes.Buffer
		err := runCheck(cfg, []string{"--ns", "doc", "--obj", "1", "--rel", "view", "--subj", "user:alice"}, &out)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(out.String(), "ALLOW") {
			t.Fatalf("expected ALLOW output, got %q", out.String())
		}
		if gotPayload["subject_namespace"] != "user" || gotPayload["subject_id"] != "alice" {
			t.Fatalf("unexpected subject payload: %#v", gotPayload)
		}
	})

	t.Run("non-200 status returns error", func(t *testing.T) {
		cfg := Config{
			NexusURL: "https://nexus.internal",
			HTTPClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					return mockHTTPResponse(http.StatusForbidden, `{"error":"forbidden"}`), nil
				}),
			},
		}
		var out bytes.Buffer
		if err := runCheck(cfg, []string{"--ns", "doc", "--obj", "1", "--rel", "view", "--subj", "alice"}, &out); err == nil {
			t.Fatal("expected error on HTTP 403")
		}
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		cfg := Config{
			NexusURL: "https://nexus.internal",
			HTTPClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					return mockHTTPResponse(http.StatusOK, `{not-json`), nil
				}),
			},
		}
		var out bytes.Buffer
		if err := runCheck(cfg, []string{"--ns", "doc", "--obj", "1", "--rel", "view", "--subj", "alice"}, &out); err == nil {
			t.Fatal("expected error on malformed JSON")
		}
	})

	t.Run("missing allowed field returns error", func(t *testing.T) {
		cfg := Config{
			NexusURL: "https://nexus.internal",
			HTTPClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					return mockHTTPResponse(http.StatusOK, `{"status":"ok"}`), nil
				}),
			},
		}
		var out bytes.Buffer
		if err := runCheck(cfg, []string{"--ns", "doc", "--obj", "1", "--rel", "view", "--subj", "alice"}, &out); err == nil {
			t.Fatal("expected error when 'allowed' boolean field is missing")
		}
	})
}

func TestRunAudit_ValidatesStatusAndJSON(t *testing.T) {
	t.Run("verify succeeds on 200 valid JSON", func(t *testing.T) {
		var gotPath string
		cfg := Config{
			ArgusURL: "https://argus.internal",
			HTTPClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					gotPath = r.URL.Path
					return mockHTTPResponse(http.StatusOK, `{"valid":true,"checked":42}`), nil
				}),
			},
		}
		var out bytes.Buffer
		if err := runAudit(cfg, []string{"verify"}, &out); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotPath != "/admin/audit/verify" {
			t.Fatalf("expected /admin/audit/verify, got %q", gotPath)
		}
		if !strings.Contains(out.String(), `"valid":true`) {
			t.Fatalf("unexpected output: %q", out.String())
		}
	})

	t.Run("list fails on non-200", func(t *testing.T) {
		cfg := Config{
			ArgusURL: "https://argus.internal",
			HTTPClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					return mockHTTPResponse(http.StatusUnauthorized, `unauthorized`), nil
				}),
			},
		}
		var out bytes.Buffer
		if err := runAudit(cfg, []string{"list"}, &out); err == nil {
			t.Fatal("expected error on HTTP 401")
		}
	})

	t.Run("verify fails on invalid JSON", func(t *testing.T) {
		cfg := Config{
			ArgusURL: "https://argus.internal",
			HTTPClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					return mockHTTPResponse(http.StatusOK, `<html>502 gateway</html>`), nil
				}),
			},
		}
		var out bytes.Buffer
		if err := runAudit(cfg, []string{"verify"}, &out); err == nil {
			t.Fatal("expected error on non-JSON body")
		}
	})

	t.Run("unknown subcommand fails", func(t *testing.T) {
		var out bytes.Buffer
		if err := runAudit(Config{}, []string{"purge"}, &out); err == nil {
			t.Fatal("expected error on unknown audit subcommand")
		}
	})
}

func TestRunApply_ValidatesFileAndReturnsNotImplemented(t *testing.T) {
	dir := t.TempDir()

	t.Run("valid YAML returns explicit not-implemented error without fake success", func(t *testing.T) {
		p := filepath.Join(dir, "policy.yaml")
		if err := os.WriteFile(p, []byte("kind: Policy\nname: test-policy\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		err := runApply(Config{}, []string{"-f", p}, &out)
		if !errors.Is(err, errApplyNotImplemented) {
			t.Fatalf("expected errApplyNotImplemented, got %v", err)
		}
		if strings.Contains(out.String(), "Success") {
			t.Fatalf("must not emit fake success output, got %q", out.String())
		}
	})

	t.Run("valid JSON returns explicit not-implemented error", func(t *testing.T) {
		p := filepath.Join(dir, "policy.json")
		if err := os.WriteFile(p, []byte(`{"kind":"Policy","name":"test"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		err := runApply(Config{}, []string{"-f", p}, &out)
		if !errors.Is(err, errApplyNotImplemented) {
			t.Fatalf("expected errApplyNotImplemented, got %v", err)
		}
	})

	t.Run("invalid JSON fails validation", func(t *testing.T) {
		p := filepath.Join(dir, "bad.json")
		if err := os.WriteFile(p, []byte(`{"kind":`), 0o600); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		err := runApply(Config{}, []string{"-f", p}, &out)
		if err == nil || errors.Is(err, errApplyNotImplemented) {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("YAML with tab indentation fails validation", func(t *testing.T) {
		p := filepath.Join(dir, "tabs.yaml")
		if err := os.WriteFile(p, []byte("kind: Policy\n\tname: bad\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		err := runApply(Config{}, []string{"-f", p}, &out)
		if err == nil || errors.Is(err, errApplyNotImplemented) {
			t.Fatalf("expected validation error for tab indentation, got %v", err)
		}
	})

	t.Run("empty YAML fails validation", func(t *testing.T) {
		p := filepath.Join(dir, "empty.yaml")
		if err := os.WriteFile(p, []byte("   \n# comment only\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		err := runApply(Config{}, []string{"-f", p}, &out)
		if err == nil || errors.Is(err, errApplyNotImplemented) {
			t.Fatalf("expected validation error for empty/comment-only YAML, got %v", err)
		}
	})

	t.Run("unsupported extension fails validation", func(t *testing.T) {
		p := filepath.Join(dir, "config.txt")
		if err := os.WriteFile(p, []byte("name: test\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		err := runApply(Config{}, []string{"-f", p}, &out)
		if err == nil || errors.Is(err, errApplyNotImplemented) {
			t.Fatalf("expected validation error for .txt extension, got %v", err)
		}
	})

	t.Run("missing file returns read error", func(t *testing.T) {
		var out bytes.Buffer
		err := runApply(Config{}, []string{"-f", filepath.Join(dir, "missing.yaml")}, &out)
		if err == nil || errors.Is(err, errApplyNotImplemented) {
			t.Fatalf("expected file read error, got %v", err)
		}
	})
}
