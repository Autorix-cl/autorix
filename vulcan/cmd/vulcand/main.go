package main

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/autorix/platform/config"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/httpx"
	platformlog "github.com/autorix/platform/log"
	"github.com/autorix/platform/metrics"
	platformpg "github.com/autorix/platform/postgres"
	"github.com/autorix/platform/registry"
	"github.com/autorix/platform/run"
	"github.com/autorix/platform/version"
	"github.com/autorix/vulcan/internal/storage/postgres"
	transport "github.com/autorix/vulcan/internal/transport/http"
	"github.com/google/uuid"
)

type appConfig struct {
	Port            string        `env:"PORT" envDefault:"4466"`
	DatabaseURL     string        `env:"DATABASE_URL" envDefault:"postgres://autorix:autorix_password@localhost:5432/autorix_vulcan?sslmode=disable"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info"`
	InstanceID      string        `env:"AUTORIX_INSTANCE_ID"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	RequestTimeout  time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
	CORSOrigins     string        `env:"CORS_ALLOWED_ORIGINS" envDefault:"*"`
	LocationURL     string        `env:"LOCATION_URL" envDefault:"https://api.autorix.io"`
}

func main() {
	startedAt := time.Now()

	var cfg appConfig
	if err := config.Load(&cfg); err != nil {
		platformlog.New(platformlog.Config{Engine: "vulcan", Level: "info"}, nil).
			Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	instanceID := cfg.InstanceID
	if instanceID == "" {
		instanceID = uuid.NewString()
	}

	logger := platformlog.New(platformlog.Config{
		Engine:     "vulcan",
		InstanceID: instanceID,
		Level:      cfg.LogLevel,
	}, nil)

	logger.Info("starting Autorix Vulcan (API Keys & Macaroons Engine)")

	ctx, cancel := run.NotifyContext()
	defer cancel()

	// 1. Database Connection
	pool, err := platformpg.Connect(ctx, cfg.DatabaseURL, platformpg.ConnectOptions{})
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	metrics.RegisterPoolStats("vulcan", pool)

	// 2. Storage and Server
	repo := postgres.NewRepository(pool)

	// 2.5 Health & Readiness
	checker := health.NewChecker()
	checker.Register("postgres", platformpg.Check(pool))
	healthHandler := health.NewHandler(checker, func() health.Info {
		return health.Info{
			Engine:     "vulcan",
			Version:    version.Version,
			BuildSHA:   version.BuildSHA,
			InstanceID: instanceID,
			StartedAt:  startedAt,
		}
	})

	server := transport.NewServer(repo, cfg.LocationURL, healthHandler)

	handler := httpx.Chain(server.Routes(),
		httpx.RequestID,
		metrics.HTTPMiddleware("vulcan"),
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

	logger.Info("Autorix Vulcan listening", "port", cfg.Port, "protocol", "REST")

	// Optional control-plane registration (Argus). No-op unless
	// AUTORIX_ARGUS_URL/AUTORIX_ENROLLMENT_TOKEN are set; never blocks or
	// fails engine startup.
	registryClient := registry.NewFromEnv("vulcan", registry.Endpoints{REST: cfg.LocationURL},
		[]string{"health.v1"},
		[]registry.Dependency{{Name: "postgres", Target: registry.PostgresDependencyTarget(cfg.DatabaseURL)}},
		logger)
	registryClient.Start(ctx)

	err = run.Run(ctx, cfg.ShutdownTimeout, logger, []run.Named{
		{
			Name: "vulcan-http",
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

	logger.Info("Autorix Vulcan stopped")
}
