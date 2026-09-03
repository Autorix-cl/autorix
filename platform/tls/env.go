package autortls

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// ServerConfigFromEnv reads server TLS configuration from standard environment variables.
// Returns (cfg, true) if TLS certificate and key files are provided.
func ServerConfigFromEnv() (ServerConfig, bool) {
	certFile := os.Getenv("AUTORIX_TLS_CERT_FILE")
	if certFile == "" {
		certFile = os.Getenv("TLS_CERT_FILE")
	}

	keyFile := os.Getenv("AUTORIX_TLS_KEY_FILE")
	if keyFile == "" {
		keyFile = os.Getenv("TLS_KEY_FILE")
	}

	caFile := os.Getenv("AUTORIX_TLS_CA_FILE")
	if caFile == "" {
		caFile = os.Getenv("TLS_CA_FILE")
	}

	requireMTLS := strings.EqualFold(os.Getenv("AUTORIX_TLS_REQUIRE_MTLS"), "true") ||
		strings.EqualFold(os.Getenv("TLS_REQUIRE_MTLS"), "true")

	if certFile == "" || keyFile == "" {
		return ServerConfig{}, false
	}

	return ServerConfig{
		CertFile:    certFile,
		KeyFile:     keyFile,
		CAFile:      caFile,
		RequireMTLS: requireMTLS,
	}, true
}

// ClientConfigFromEnv reads client TLS configuration from standard environment variables.
func ClientConfigFromEnv() (ClientConfig, bool) {
	certFile := os.Getenv("AUTORIX_TLS_CLIENT_CERT_FILE")
	if certFile == "" {
		certFile = os.Getenv("TLS_CLIENT_CERT_FILE")
	}

	keyFile := os.Getenv("AUTORIX_TLS_CLIENT_KEY_FILE")
	if keyFile == "" {
		keyFile = os.Getenv("TLS_CLIENT_KEY_FILE")
	}

	caFile := os.Getenv("AUTORIX_TLS_CA_FILE")
	if caFile == "" {
		caFile = os.Getenv("TLS_CA_FILE")
	}

	skipVerify := strings.EqualFold(os.Getenv("AUTORIX_TLS_INSECURE_SKIP_VERIFY"), "true") ||
		strings.EqualFold(os.Getenv("TLS_INSECURE_SKIP_VERIFY"), "true")

	if certFile == "" && caFile == "" {
		return ClientConfig{}, false
	}

	return ClientConfig{
		CertFile:           certFile,
		KeyFile:            keyFile,
		CAFile:             caFile,
		InsecureSkipVerify: skipVerify,
	}, true
}

// NewClientTransport builds an http.Transport with mutual TLS support.
func NewClientTransport(cfg ClientConfig) (*http.Transport, error) {
	tlsConfig, err := NewClientTLSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("build client tls config: %w", err)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig
	return transport, nil
}
