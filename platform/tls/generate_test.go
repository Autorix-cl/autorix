package autortls

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateMeshPKI(t *testing.T) {
	tmpDir := t.TempDir()
	services := []string{"nexus", "vulcan", "ego", "console"}

	err := GenerateMeshPKI(tmpDir, services)
	if err != nil {
		t.Fatalf("GenerateMeshPKI failed: %v", err)
	}

	// Verify CA exists
	caCertFile := filepath.Join(tmpDir, "ca.crt")
	caKeyFile := filepath.Join(tmpDir, "ca.key")
	if _, err := os.Stat(caCertFile); err != nil {
		t.Fatalf("ca.crt missing: %v", err)
	}
	if _, err := os.Stat(caKeyFile); err != nil {
		t.Fatalf("ca.key missing: %v", err)
	}

	// Verify service certs and keys exist
	for _, svc := range services {
		certFile := filepath.Join(tmpDir, svc+".crt")
		keyFile := filepath.Join(tmpDir, svc+".key")
		if _, err := os.Stat(certFile); err != nil {
			t.Fatalf("%s.crt missing: %v", svc, err)
		}
		if _, err := os.Stat(keyFile); err != nil {
			t.Fatalf("%s.key missing: %v", svc, err)
		}

		// Verify cert loads cleanly
		_, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			t.Fatalf("failed to load keypair for %s: %v", svc, err)
		}
	}
}

func TestEndToEndMTLSHttpsConnection(t *testing.T) {
	tmpDir := t.TempDir()
	err := GenerateMeshPKI(tmpDir, []string{"server", "authorized-client", "unauthorized-client"})
	if err != nil {
		t.Fatalf("generate pki: %v", err)
	}

	caFile := filepath.Join(tmpDir, "ca.crt")
	serverCert := filepath.Join(tmpDir, "server.crt")
	serverKey := filepath.Join(tmpDir, "server.key")
	authClientCert := filepath.Join(tmpDir, "authorized-client.crt")
	authClientKey := filepath.Join(tmpDir, "authorized-client.key")

	serverTLS, err := NewServerTLSConfig(ServerConfig{
		CertFile:    serverCert,
		KeyFile:     serverKey,
		CAFile:      caFile,
		RequireMTLS: true,
	})
	if err != nil {
		t.Fatalf("server tls config: %v", err)
	}

	// Handlers with mTLS verification middleware
	mux := http.NewServeMux()
	mux.Handle("/secure", RequirePeerIdentity("authorized-client")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity := PeerIdentity(r.Context())
		spiffe := PeerSpiffeID(r.Context())
		w.Header().Set("X-Peer-Identity", identity)
		w.Header().Set("X-Peer-Spiffe", spiffe)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("secure-data"))
	})))

	ts := httptest.NewUnstartedServer(mux)
	ts.TLS = serverTLS
	ts.StartTLS()
	defer ts.Close()

	// 1. Authorized client connects with valid client cert
	clientTLS, err := NewClientTLSConfig(ClientConfig{
		CertFile: authClientCert,
		KeyFile:  authClientKey,
		CAFile:   caFile,
	})
	if err != nil {
		t.Fatalf("client tls config: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: clientTLS,
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(ts.URL + "/secure")
	if err != nil {
		t.Fatalf("authorized request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Peer-Identity") != "authorized-client" {
		t.Fatalf("expected peer identity authorized-client, got %s", resp.Header.Get("X-Peer-Identity"))
	}
	expectedSpiffe := "spiffe://autorix.internal/service/authorized-client"
	if resp.Header.Get("X-Peer-Spiffe") != expectedSpiffe {
		t.Fatalf("expected spiffe %s, got %s", expectedSpiffe, resp.Header.Get("X-Peer-Spiffe"))
	}

	// 2. Client connecting WITHOUT cert must fail TLS handshake
	noCertTLS, err := NewClientTLSConfig(ClientConfig{CAFile: caFile})
	if err != nil {
		t.Fatalf("no cert tls config: %v", err)
	}
	unauthenticatedClient := &http.Client{
		Transport: &http.Transport{TLSClientConfig: noCertTLS},
		Timeout:   2 * time.Second,
	}

	_, err = unauthenticatedClient.Get(ts.URL + "/secure")
	if err == nil {
		t.Fatalf("expected TLS handshake failure when connecting without client cert, got nil")
	}
}
