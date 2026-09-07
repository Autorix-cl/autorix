package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/autorix/platform/httpx"
)

func TestLoadJWTAuthenticator(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&key.PublicKey)}), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		c     cfg
		valid bool
	}{
		{name: "valid pinned key", c: cfg{JWTPublicKeyFile: path, JWTIssuer: "issuer", JWTAudience: "api"}, valid: true},
		{name: "missing trust", c: cfg{}},
		{name: "ambiguous trust", c: cfg{JWTPublicKeyFile: path, JWTJWKSURL: "https://example.com/jwks"}},
		{name: "missing issuer", c: cfg{JWTPublicKeyFile: path, JWTAudience: "api"}},
		{name: "missing audience", c: cfg{JWTPublicKeyFile: path, JWTIssuer: "issuer"}},
		{name: "missing key file", c: cfg{JWTPublicKeyFile: path + ".missing", JWTIssuer: "issuer", JWTAudience: "api"}},
		{name: "insecure endpoint", c: cfg{JWTJWKSURL: "http://example.com/jwks", JWTIssuer: "issuer", JWTAudience: "api"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadJWTAuthenticator(context.Background(), tc.c)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
		})
	}
}

func TestAdminCORSDefaultsToDeny(t *testing.T) {
	handler := httpx.CORS(httpx.CORSConfig{AllowedOrigins: []string{""}})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "http://aegis/admin/rules", nil)
	req.Header.Set("Origin", "https://attacker.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unset CORS origin allowed %q", got)
	}
}
