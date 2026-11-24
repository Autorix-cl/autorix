package autortls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func generateTestCertAndKey(t *testing.T, commonName string, isCA bool) (certPEM, keyPEM []byte) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Autorix Test"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}

	if isCA {
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create cert: %v", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return certPEM, keyPEM
}

func TestServerAndClientMTLS(t *testing.T) {
	tmpDir := t.TempDir()

	caCert, _ := generateTestCertAndKey(t, "Autorix CA", true)
	serverCert, serverKey := generateTestCertAndKey(t, "argus.autorix.internal", false)
	clientCert, clientKey := generateTestCertAndKey(t, "ego-instance-1", false)

	caFile := filepath.Join(tmpDir, "ca.pem")
	serverCertFile := filepath.Join(tmpDir, "server.pem")
	serverKeyFile := filepath.Join(tmpDir, "server.key")
	clientCertFile := filepath.Join(tmpDir, "client.pem")
	clientKeyFile := filepath.Join(tmpDir, "client.key")

	for name, content := range map[string][]byte{
		caFile:         caCert,
		serverCertFile: serverCert,
		serverKeyFile:  serverKey,
		clientCertFile: clientCert,
		clientKeyFile:  clientKey,
	} {
		if err := os.WriteFile(name, content, 0600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	serverTLS, err := NewServerTLSConfig(ServerConfig{
		CertFile:    serverCertFile,
		KeyFile:     serverKeyFile,
		CAFile:      caFile,
		RequireMTLS: true,
	})
	if err != nil {
		t.Fatalf("failed to create server TLS config: %v", err)
	}
	if serverTLS == nil {
		t.Fatal("expected serverTLS config, got nil")
	}

	clientTLS, err := NewClientTLSConfig(ClientConfig{
		CertFile: clientCertFile,
		KeyFile:  clientKeyFile,
		CAFile:   caFile,
	})
	if err != nil {
		t.Fatalf("failed to create client TLS config: %v", err)
	}
	if clientTLS == nil {
		t.Fatal("expected clientTLS config, got nil")
	}
}

func TestVerifyPeerIdentity(t *testing.T) {
	certPEM, _ := generateTestCertAndKey(t, "ego-prod-1", false)
	block, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse cert: %v", err)
	}

	commonName := cert.Subject.CommonName

	// Match
	if err := VerifyPeerCertificateSubject(commonName, "ego-prod-1"); err != nil {
		t.Errorf("expected subject match, got %v", err)
	}

	// Mismatch
	if err := VerifyPeerCertificateSubject(commonName, "vulcan-prod-2"); err == nil {
		t.Error("expected error on subject mismatch, got nil")
	}
}

func TestExtractServiceFromSPIFFE(t *testing.T) {
	tests := []struct {
		uri      string
		expected string
		ok       bool
	}{
		{"spiffe://autorix.internal/service/nexus", "nexus", true},
		{"spiffe://autorix.internal/service/EGO", "ego", true},
		{"spiffe://autorix.internal/service/vulcan/", "vulcan", true},
		{"spiffe://other.domain/service/nexus", "", false},
		{"invalid-uri", "", false},
	}

	for _, tc := range tests {
		svc, ok := ExtractServiceFromSPIFFE(tc.uri)
		if ok != tc.ok || svc != tc.expected {
			t.Errorf("ExtractServiceFromSPIFFE(%q) = (%q, %v); expected (%q, %v)", tc.uri, svc, ok, tc.expected, tc.ok)
		}
	}
}

func TestNewClientTLSConfig_RejectsPartialKeyPair(t *testing.T) {
	if _, err := NewClientTLSConfig(ClientConfig{CertFile: "only-cert.pem"}); err == nil {
		t.Fatal("expected error when only CertFile is provided")
	}
	if _, err := NewClientTLSConfig(ClientConfig{KeyFile: "only-key.pem"}); err == nil {
		t.Fatal("expected error when only KeyFile is provided")
	}
}

func TestRequirePeerIdentity_InspectsAllURIsAndNoCNFallbackWhenSpiffePresent(t *testing.T) {
	mustURL := func(raw string) *url.URL {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("parse url: %v", err)
		}
		return u
	}

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 1. Disallowed SPIFFE URI must NOT fall back to an allowed CommonName.
	reqDisallowedSpiffe := httptest.NewRequest(http.MethodGet, "/", nil)
	reqDisallowedSpiffe.TLS = &tls.ConnectionState{
		VerifiedChains: [][]*x509.Certificate{{
			{
				Subject: pkix.Name{CommonName: "ego"},
				URIs:    []*url.URL{mustURL("spiffe://autorix.internal/service/evil")},
			},
		}},
	}
	rec1 := httptest.NewRecorder()
	RequirePeerIdentity("ego")(okHandler).ServeHTTP(rec1, reqDisallowedSpiffe)
	if rec1.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when SPIFFE URI is disallowed even if CN matches, got %d", rec1.Code)
	}

	// 2. Allowed SPIFFE URI in second URI slot must be accepted.
	reqSecondURI := httptest.NewRequest(http.MethodGet, "/", nil)
	reqSecondURI.TLS = &tls.ConnectionState{
		VerifiedChains: [][]*x509.Certificate{{
			{
				Subject: pkix.Name{CommonName: "other"},
				URIs: []*url.URL{
					mustURL("https://example.com/meta"),
					mustURL("spiffe://autorix.internal/service/ego"),
				},
			},
		}},
	}
	rec2 := httptest.NewRecorder()
	RequirePeerIdentity("ego")(okHandler).ServeHTTP(rec2, reqSecondURI)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 when allowed SPIFFE URI is in non-first URI slot, got %d", rec2.Code)
	}

	// 3. No SPIFFE URI present: falls back to CommonName cleanly.
	reqCNOnly := httptest.NewRequest(http.MethodGet, "/", nil)
	reqCNOnly.TLS = &tls.ConnectionState{
		VerifiedChains: [][]*x509.Certificate{{
			{
				Subject: pkix.Name{CommonName: "ego"},
			},
		}},
	}
	rec3 := httptest.NewRecorder()
	RequirePeerIdentity("ego")(okHandler).ServeHTTP(rec3, reqCNOnly)
	if rec3.Code != http.StatusOK {
		t.Fatalf("expected 200 when no SPIFFE URI is present and CN matches, got %d", rec3.Code)
	}
}
