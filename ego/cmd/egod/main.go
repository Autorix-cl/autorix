package main

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/autorix/ego/internal/credential"
	"github.com/autorix/ego/internal/session"
	"github.com/autorix/ego/internal/storage/postgres"
	transport "github.com/autorix/ego/internal/transport/http"
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
	Port            string        `env:"PORT" envDefault:"4433"`
	DatabaseURL     string        `env:"DATABASE_URL" envDefault:"postgres://autorix:autorix_password@localhost:5432/autorix_ego?sslmode=disable"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info"`
	InstanceID      string        `env:"AUTORIX_INSTANCE_ID"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	RequestTimeout  time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
	CORSOrigins     string        `env:"CORS_ALLOWED_ORIGINS" envDefault:"*"`
	// SecureCookies must be true whenever ego is reachable over TLS — the
	// default. Set to false only for plain-HTTP local development.
	SecureCookies bool `env:"SECURE_COOKIES" envDefault:"true"`
}

func main() {
	startedAt := time.Now()

	var cfg appConfig
	if err := config.Load(&cfg); err != nil {
		platformlog.New(platformlog.Config{Engine: "ego", Level: "info"}, nil).
			Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	instanceID := cfg.InstanceID
	if instanceID == "" {
		instanceID = uuid.NewString()
	}

	logger := platformlog.New(platformlog.Config{
		Engine:     "ego",
		InstanceID: instanceID,
		Level:      cfg.LogLevel,
	}, nil)

	logger.Info("starting Autorix Ego (Identity Engine)")

	ctx, cancel := run.NotifyContext()
	defer cancel()

	// 1. Database Connection
	pool, err := platformpg.Connect(ctx, cfg.DatabaseURL, platformpg.ConnectOptions{})
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	metrics.RegisterPoolStats("ego", pool)

	// 2. Initialize Domain & Storage
	repo := postgres.NewRepository(pool)
	hasher := credential.NewHasher(nil)
	sessionManager := session.NewManager(30 * 24 * time.Hour)

	// 2.5 Health & Readiness
	checker := health.NewChecker()
	checker.Register("postgres", platformpg.Check(pool))
	healthHandler := health.NewHandler(checker, func() health.Info {
		return health.Info{
			Engine:     "ego",
			Version:    version.Version,
			BuildSHA:   version.BuildSHA,
			InstanceID: instanceID,
			StartedAt:  startedAt,
		}
	})

	// 3. HTTP Server
	server := transport.NewServer(repo, hasher, sessionManager, healthHandler, cfg.SecureCookies)

	handler := httpx.Chain(server.Routes(),
		httpx.RequestID,
		metrics.HTTPMiddleware("ego"),
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

	logger.Info("Autorix Ego listening", "port", cfg.Port, "protocol", "HTTP REST")

	// 4. Optional control-plane registration (Argus). No-op unless
	// AUTORIX_ARGUS_URL/AUTORIX_ENROLLMENT_TOKEN are set; never blocks or
	// fails engine startup.
	registryClient := registry.NewFromEnv("ego", registry.Endpoints{REST: "http://localhost:" + cfg.Port},
		[]string{"health.v1"},
		[]registry.Dependency{{Name: "postgres", Target: registry.PostgresDependencyTarget(cfg.DatabaseURL)}},
		logger)
	registryClient.Start(ctx)

	err = run.Run(ctx, cfg.ShutdownTimeout, logger, []run.Named{
		{
			Name: "ego-http",
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

	logger.Info("Autorix Ego stopped")
}
