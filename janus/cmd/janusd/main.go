package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/autorix/janus/internal/jwks"
	"github.com/autorix/janus/internal/oauth2"
	"github.com/autorix/janus/internal/storage/postgres"
	transport "github.com/autorix/janus/internal/transport/http"
	"github.com/autorix/platform/config"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/httpx"
	platformlog "github.com/autorix/platform/log"
	"github.com/autorix/platform/metrics"
	platformpg "github.com/autorix/platform/postgres"
	"github.com/autorix/platform/registry"
	"github.com/autorix/platform/run"
	"github.com/autorix/platform/version"
	"github.com/google/uuid"
)

type appConfig struct {
	AdminHost         string        `env:"ADMIN_HOST" envDefault:"127.0.0.1"`
	AdminPort         string        `env:"ADMIN_PORT" envDefault:"4445"`
	Port              string        `env:"PORT" envDefault:"4444"`
	DatabaseURL       string        `env:"DATABASE_URL" envDefault:"postgres://autorix:autorix_password@localhost:5432/autorix_janus?sslmode=disable"`
	LogLevel          string        `env:"LOG_LEVEL" envDefault:"info"`
	InstanceID        string        `env:"AUTORIX_INSTANCE_ID"`
	ShutdownTimeout   time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	RequestTimeout    time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
	CORSOrigins       string        `env:"CORS_ALLOWED_ORIGINS"`
	IssuerURL         string        `env:"ISSUER_URL" envDefault:"http://localhost:4444"`
	KeyReloadInterval time.Duration `env:"KEY_RELOAD_INTERVAL" envDefault:"30s"`
}

func main() {
	startedAt := time.Now()

	var cfg appConfig
	if err := config.Load(&cfg); err != nil {
		platformlog.New(platformlog.Config{Engine: "janus", Level: "info"}, nil).
			Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	if err := cfg.validate(); err != nil {
		platformlog.New(platformlog.Config{Engine: "janus", Level: "info"}, nil).
			Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	instanceID := cfg.InstanceID
	if instanceID == "" {
		instanceID = uuid.NewString()
	}

	logger := platformlog.New(platformlog.Config{
		Engine:     "janus",
		InstanceID: instanceID,
		Level:      cfg.LogLevel,
	}, nil)

	logger.Info("starting Autorix Janus (OAuth2 & OIDC Server)")

	ctx, cancel := run.NotifyContext()
	defer cancel()

	// 1. Database Connection
	pool, err := platformpg.Connect(ctx, cfg.DatabaseURL, platformpg.ConnectOptions{})
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	metrics.RegisterPoolStats("janus", pool)

	// 2. Load the shared persistent signing-key set before accepting traffic.
	// This prevents restarts or additional replicas from minting unrelated JWTs.
	repo := postgres.NewRepository(pool)
	keyManager, err := jwks.NewPersistentKeyManager(ctx, repo)
	if err != nil {
		logger.Error("failed to initialize JWKS Key Manager", "error", err)
		os.Exit(1)
	}

	engine := oauth2.NewEngine(cfg.IssuerURL, keyManager)
	keyReloadRunner, err := newKeyReloadRunner(keyManager, cfg.KeyReloadInterval, logger)
	if err != nil {
		logger.Error("failed to configure persistent signing-key reload", "error", err)
		os.Exit(1)
	}

	// 2.5 Health & Readiness
	checker := health.NewChecker()
	checker.Register("postgres", platformpg.Check(pool))
	healthHandler := health.NewHandler(checker, func() health.Info {
		return health.Info{
			Engine:     "janus",
			Version:    version.Version,
			BuildSHA:   version.BuildSHA,
			InstanceID: instanceID,
			StartedAt:  startedAt,
		}
	})

	// 3. HTTP Server
	server := transport.NewServer(cfg.IssuerURL, repo, keyManager, engine, healthHandler)

	handler := httpx.Chain(server.Routes(),
		httpx.RequestID,
		metrics.HTTPMiddleware("janus"),
		httpx.Recover(logger),
		httpx.AccessLog(logger),
		httpx.Timeout(cfg.RequestTimeout),
		httpx.CORS(httpx.CORSConfig{AllowedOrigins: corsAllowedOrigins(cfg.CORSOrigins)}),
	)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	adminServer := &http.Server{
		Addr: net.JoinHostPort(cfg.AdminHost, cfg.AdminPort),
		Handler: httpx.Chain(server.AdminRoutes(),
			httpx.RequestID,
			metrics.HTTPMiddleware("janus"),
			httpx.Recover(logger),
			httpx.AccessLog(logger),
			httpx.Timeout(cfg.RequestTimeout),
		),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Bind both listeners before serving either, so an admin bind failure does
	// not leave a partially started public service.
	listeners, err := bindHTTPServers(httpServer, adminServer)
	if err != nil {
		logger.Error("failed to bind HTTP listeners", "error", err)
		os.Exit(1)
	}
	logger.Info("Janus admin listener requires a private network", "address", adminServer.Addr)

	logger.Info("Autorix Janus listening", "port", cfg.Port, "protocol", "OAuth2/OIDC")
	logger.Info("JWKS exposed", "url", cfg.IssuerURL+"/.well-known/jwks.json")

	// Optional control-plane registration (Argus). No-op unless
	// AUTORIX_ARGUS_URL/AUTORIX_ENROLLMENT_TOKEN are set; never blocks or
	// fails engine startup.
	registryClient := registry.NewFromEnv("janus", registry.Endpoints{REST: cfg.IssuerURL},
		[]string{"health.v1"},
		[]registry.Dependency{{Name: "postgres", Target: registry.PostgresDependencyTarget(cfg.DatabaseURL)}},
		logger)
	registryClient.Start(ctx)

	err = run.Run(ctx, cfg.ShutdownTimeout, logger, []run.Named{
		httpRunner("janus-http", httpServer, listeners[0]),
		httpRunner("janus-admin", adminServer, listeners[1]),
		{
			Name: "janus-key-reload",
			Serve: func() error {
				return keyReloadRunner.Serve(ctx)
			},
			Shutdown: keyReloadRunner.Shutdown,
		},
		{
			Name:     "argus-registry",
			Serve:    func() error { <-ctx.Done(); return run.ErrServerClosed },
			Shutdown: registryClient.Stop,
		},
	})
	if err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}

	logger.Info("Autorix Janus stopped")
}

// bindHTTPServers closes earlier listeners if any bind fails.
func bindHTTPServers(servers ...*http.Server) ([]net.Listener, error) {
	listeners := make([]net.Listener, 0, len(servers))
	for _, server := range servers {
		listener, err := net.Listen("tcp", server.Addr)
		if err != nil {
			for _, opened := range listeners {
				_ = opened.Close()
			}
			return nil, fmt.Errorf("listen on %s: %w", server.Addr, err)
		}
		listeners = append(listeners, listener)
	}
	return listeners, nil
}

func httpRunner(name string, server *http.Server, listener net.Listener) run.Named {
	return run.Named{
		Name: name,
		Serve: func() error {
			defer func() { _ = listener.Close() }()
			if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return run.ErrServerClosed
		},
		Shutdown: server.Shutdown,
	}
}

func (cfg appConfig) validate() error {
	if strings.TrimSpace(cfg.AdminHost) == "" || strings.TrimSpace(cfg.AdminPort) == "" {
		return errors.New("ADMIN_HOST and ADMIN_PORT must not be empty")
	}
	if cfg.KeyReloadInterval <= 0 {
		return errors.New("KEY_RELOAD_INTERVAL must be positive")
	}
	return nil
}

// corsAllowedOrigins returns only explicitly configured origins. An empty
// configuration intentionally denies browser cross-origin access rather than
// silently falling back to a wildcard.
func corsAllowedOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}
