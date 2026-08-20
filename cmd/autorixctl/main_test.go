package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDoRequest_InjectsHeadersAndExecutes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Errorf("expected bearer auth, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestGetEnv_Fallback(t *testing.T) {
	os.Unsetenv("TEST_AUTORIX_VAR")
	val := getEnv("TEST_AUTORIX_VAR", "default_val")
	if val != "default_val" {
		t.Errorf("expected default_val, got %s", val)
	}

	os.Setenv("TEST_AUTORIX_VAR", "custom")
	defer os.Unsetenv("TEST_AUTORIX_VAR")
	val = getEnv("TEST_AUTORIX_VAR", "default_val")
	if val != "custom" {
		t.Errorf("expected custom, got %s", val)
	}
}
