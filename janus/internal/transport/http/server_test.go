package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autorix/janus/internal/jwks"
	"github.com/autorix/janus/internal/oauth2"
)

func TestDiscoveryAndJWKS(t *testing.T) {
	km, err := jwks.NewKeyManager()
	if err != nil {
		t.Fatalf("failed to create KeyManager: %v", err)
	}

	engine := oauth2.NewEngine("http://localhost:4444", km)
	server := NewServer("http://localhost:4444", nil, km, engine)
	router := server.Routes()

	// 1. Test OpenID Discovery
	req := httptest.NewRequest("GET", "/.well-known/openid-configuration", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK from discovery, got %d", rec.Code)
	}

	var discovery map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &discovery); err != nil {
		t.Fatalf("failed to parse discovery JSON: %v", err)
	}

	if discovery["issuer"] != "http://localhost:4444" {
		t.Errorf("expected issuer 'http://localhost:4444', got %v", discovery["issuer"])
	}

	// 2. Test JWKS endpoint
	reqJWKS := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	recJWKS := httptest.NewRecorder()
	router.ServeHTTP(recJWKS, reqJWKS)

	if recJWKS.Code != http.StatusOK {
		t.Errorf("expected 200 OK from JWKS, got %d", recJWKS.Code)
	}

	var jwksResp map[string][]interface{}
	if err := json.Unmarshal(recJWKS.Body.Bytes(), &jwksResp); err != nil {
		t.Fatalf("failed to parse JWKS JSON: %v", err)
	}

	if len(jwksResp["keys"]) != 1 {
		t.Errorf("expected 1 key in JWKS, got %d", len(jwksResp["keys"]))
	}
}
