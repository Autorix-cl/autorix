package autortls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

// ServerConfig defines the TLS parameters for an HTTPS/mTLS server.
type ServerConfig struct {
	CertFile    string `env:"AUTORIX_TLS_CERT_FILE"`
	KeyFile     string `env:"AUTORIX_TLS_KEY_FILE"`
	CAFile      string `env:"AUTORIX_TLS_CA_FILE"`
	RequireMTLS bool   `env:"AUTORIX_TLS_REQUIRE_MTLS"`
}

// ClientConfig defines the TLS parameters for an HTTPS/mTLS client.
type ClientConfig struct {
	CertFile           string `env:"AUTORIX_TLS_CLIENT_CERT_FILE"`
	KeyFile            string `env:"AUTORIX_TLS_CLIENT_KEY_FILE"`
	CAFile             string `env:"AUTORIX_TLS_CA_FILE"`
	InsecureSkipVerify bool   `env:"AUTORIX_TLS_INSECURE_SKIP_VERIFY"`
}

// NewServerTLSConfig builds a crypto/tls.Config for server mTLS.
func NewServerTLSConfig(cfg ServerConfig) (*tls.Config, error) {
	if cfg.CertFile == "" || cfg.KeyFile == "" {
		return nil, fmt.Errorf("both CertFile and KeyFile are required for server TLS")
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load server key pair: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	if cfg.RequireMTLS || cfg.CAFile != "" {
		if cfg.CAFile == "" {
			return nil, fmt.Errorf("CAFile is required when RequireMTLS is enabled")
		}

		caPEM, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}

		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("failed to append CA certificates to pool")
		}

		tlsConfig.ClientCAs = caPool
		if cfg.RequireMTLS {
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
		}
	}

	return tlsConfig, nil
}

// NewClientTLSConfig builds a crypto/tls.Config for client mTLS.
func NewClientTLSConfig(cfg ClientConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	if cfg.CAFile != "" {
		caPEM, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}

		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("failed to append CA certificates to pool")
		}
		tlsConfig.RootCAs = caPool
	}

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client key pair: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// VerifyPeerCertificateSubject validates that the client cert CommonName matches the claimed instance/engine identity.
func VerifyPeerCertificateSubject(peerCommonName, expectedIdentity string) error {
	if peerCommonName == "" {
		return fmt.Errorf("peer presented no client certificate CommonName")
	}
	if peerCommonName != expectedIdentity {
		return fmt.Errorf("peer certificate subject %q does not match expected identity %q", peerCommonName, expectedIdentity)
	}
	return nil
}

// ConfigureHTTPServer attaches server TLS configuration if environment variables are set.
func ConfigureHTTPServer(server *http.Server, logger *slog.Logger, serviceName string) (bool, error) {
	cfg, ok := ServerConfigFromEnv()
	if !ok {
		return false, nil
	}

	tlsConfig, err := NewServerTLSConfig(cfg)
	if err != nil {
		return false, fmt.Errorf("configure mTLS for %s: %w", serviceName, err)
	}

	server.TLSConfig = tlsConfig
	if logger != nil {
		logger.Info("mTLS security mesh active", "service", serviceName, "require_client_cert", cfg.RequireMTLS)
	}
	return true, nil
}

// ServeHTTP serves HTTPS if server.TLSConfig != nil, or HTTP otherwise.
func ServeHTTP(server *http.Server) error {
	if server.TLSConfig != nil {
		if err := server.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
