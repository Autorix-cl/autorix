package autortls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// CertPair holds the PEM-encoded certificate and private key.
type CertPair struct {
	CertPEM []byte
	KeyPEM  []byte
}

// GenerateCA creates a self-signed ECDSA P-256 Root Certificate Authority.
func GenerateCA(commonName string, validity time.Duration) (*CertPair, *x509.Certificate, *ecdsa.PrivateKey, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("generate CA private key: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("generate CA serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Autorix Internal Security Mesh"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(validity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create CA certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal CA private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	return &CertPair{CertPEM: certPEM, KeyPEM: keyPEM}, template, priv, nil
}

// GenerateServiceCert creates a dual-use (server + client) certificate signed by the Root CA.
func GenerateServiceCert(
	serviceName string,
	caCert *x509.Certificate,
	caPriv *ecdsa.PrivateKey,
	validity time.Duration,
) (*CertPair, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate service private key: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("generate service serial number: %w", err)
	}

	spiffeURI, _ := url.Parse(fmt.Sprintf("spiffe://autorix.internal/service/%s", serviceName))

	dnsNames := []string{
		serviceName,
		fmt.Sprintf("autorix-%s", serviceName),
		"localhost",
	}

	ips := []net.IP{
		net.ParseIP("127.0.0.1"),
		net.IPv6loopback,
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   serviceName,
			Organization: []string{"Autorix Service Mesh"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(validity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              dnsNames,
		IPAddresses:           ips,
		URIs:                  []*url.URL{spiffeURI},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, caCert, &priv.PublicKey, caPriv)
	if err != nil {
		return nil, fmt.Errorf("create service certificate for %s: %w", serviceName, err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal service private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	return &CertPair{CertPEM: certPEM, KeyPEM: keyPEM}, nil
}

// GenerateMeshPKI generates a complete Root CA and service certificates into targetDir.
func GenerateMeshPKI(targetDir string, services []string) error {
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return fmt.Errorf("create target dir: %w", err)
	}

	caPair, caCert, caPriv, err := GenerateCA("Autorix Mesh Root CA", 365*24*time.Hour)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(targetDir, "ca.crt"), caPair.CertPEM, 0644); err != nil {
		return fmt.Errorf("write ca.crt: %w", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "ca.key"), caPair.KeyPEM, 0600); err != nil {
		return fmt.Errorf("write ca.key: %w", err)
	}

	for _, svc := range services {
		svcPair, err := GenerateServiceCert(svc, caCert, caPriv, 90*24*time.Hour)
		if err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(targetDir, fmt.Sprintf("%s.crt", svc)), svcPair.CertPEM, 0644); err != nil {
			return fmt.Errorf("write %s.crt: %w", svc, err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, fmt.Sprintf("%s.key", svc)), svcPair.KeyPEM, 0600); err != nil {
			return fmt.Errorf("write %s.key: %w", svc, err)
		}
	}

	return nil
}
