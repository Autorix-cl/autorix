package main

import (
	"errors"
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
	Port            string        `env:"PORT" envDefault:"4444"`
	DatabaseURL     string        `env:"DATABASE_URL" envDefault:"postgres://autorix:autorix_password@localhost:5432/autorix_janus?sslmode=disable"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info"`
	InstanceID      string        `env:"AUTORIX_INSTANCE_ID"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	RequestTimeout  time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
	CORSOrigins     string        `env:"CORS_ALLOWED_ORIGINS" envDefault:"*"`
	IssuerURL       string        `env:"ISSUER_URL" envDefault:"http://localhost:4444"`
}

func main() {
	startedAt := time.Now()

	var cfg appConfig
	if err := config.Load(&cfg); err != nil {
		platformlog.New(platformlog.Config{Engine: "janus", Level: "info"}, nil).
			Error("failed to load configuration", "error", err)
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

	// 2. Initialize Key Manager & OAuth2 Engine
	keyManager, err := jwks.NewKeyManager()
	if err != nil {
		logger.Error("failed to initialize JWKS Key Manager", "error", err)
		os.Exit(1)
	}

	engine := oauth2.NewEngine(cfg.IssuerURL, keyManager)
	repo := postgres.NewRepository(pool)

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
		httpx.CORS(httpx.CORSConfig{AllowedOrigins: strings.Split(cfg.CORSOrigins, ",")}),
	)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

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
		{
			Name: "janus-http",
			Serve: func() error {
				if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					return err
				}
				return run.ErrServerClosed
			},
			Shutdown: httpServer.Shutdown,
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
