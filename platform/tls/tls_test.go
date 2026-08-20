package autortls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
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

	os.WriteFile(caFile, caCert, 0600)
	os.WriteFile(serverCertFile, serverCert, 0600)
	os.WriteFile(serverKeyFile, serverKey, 0600)
	os.WriteFile(clientCertFile, clientCert, 0600)
	os.WriteFile(clientKeyFile, clientKey, 0600)

	serverTLS, err := NewServerTLSConfig(ServerConfig{
		CertFile:   serverCertFile,
		KeyFile:    serverKeyFile,
		CAFile:     caFile,
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
