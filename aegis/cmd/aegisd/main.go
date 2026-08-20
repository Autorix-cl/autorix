package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/autorix/aegis/internal/authenticator"
	"github.com/autorix/aegis/internal/authorizer"
	"github.com/autorix/aegis/internal/core"
	"github.com/autorix/aegis/internal/mutator"
	"github.com/autorix/aegis/internal/proxy"
	"github.com/autorix/aegis/internal/rule"
	"github.com/autorix/aegis/internal/storage/postgres"
	httpTransport "github.com/autorix/aegis/internal/transport/http"
	"github.com/autorix/platform/config"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/httpx"
	platformlog "github.com/autorix/platform/log"
	"github.com/autorix/platform/metrics"
	platformpg "github.com/autorix/platform/postgres"
	"github.com/autorix/platform/registry"
	"github.com/autorix/platform/run"
	"github.com/autorix/platform/version"
)

// cfg is aegis's environment configuration, loaded via platform/config.
type cfg struct {
	RulesPath       string        `env:"RULES_PATH"`
	DatabaseURL     string        `env:"DATABASE_URL"`
	AdminPort       string        `env:"ADMIN_PORT" envDefault:"4456"`
	Port            string        `env:"PORT" envDefault:"4455"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info"`
	InstanceID      string        `env:"AUTORIX_INSTANCE_ID"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	RequestTimeout  time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
	CORSOrigins     string        `env:"CORS_ORIGINS" envDefault:"*"`
}

func main() {
	var c cfg
	if err := config.Load(&c); err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}

	startedAt := time.Now()

	instanceID := c.InstanceID
	if instanceID == "" {
		if hostname, hostErr := os.Hostname(); hostErr == nil {
			instanceID = hostname
		}
	}

	logger := platformlog.New(platformlog.Config{
		Engine:     "aegis",
		InstanceID: instanceID,
		Level:      c.LogLevel,
	}, nil)

	logger.Info("starting autorix aegis (zero trust access proxy)")

	ctx, stop := run.NotifyContext()
	defer stop()

	checker := health.NewChecker()

	// 1. Initialize Security Rules Store (Postgres with file fallback)
	var store rule.Store
	if c.DatabaseURL != "" {
		pool, err := platformpg.Connect(ctx, c.DatabaseURL, platformpg.ConnectOptions{})
		if err != nil {
			logger.Error("failed to connect to postgres", "error", err)
			os.Exit(1)
		}
		defer pool.Close()

		metrics.RegisterPostgresPool(pool, "aegis")

		pgStore, err := postgres.NewPostgresStore(ctx, pool)
		if err != nil {
			logger.Error("failed to initialize postgres rules store", "error", err)
			os.Exit(1)
		}
		store = pgStore
		checker.Register("postgres", platformpg.Check(pool))
	} else {
		rulesPath := c.RulesPath
		if rulesPath == "" {
			rulesPath = "rules/default.rules.yaml"
		}

		fileStore, err := rule.NewStore(rulesPath)
		if err != nil {
			logger.Error("failed to load rules", "error", err)
			os.Exit(1)
		}
		store = fileStore
	}

	// 2. Initialize Pipeline Handlers
	authenticators := []core.Authenticator{
		authenticator.NewJWTAuthenticator(nil), // Unverified/local parser or keyFunc
		&authenticator.AnonymousAuthenticator{},
		&authenticator.NoopAuthenticator{},
	}

	authorizers := []core.Authorizer{
		&authorizer.AllowAuthorizer{},
		&authorizer.DenyAuthorizer{},
		authorizer.NewNexusAuthorizer(),
	}

	mutators := []core.Mutator{
		mutator.NewHeaderMutator(),
		&mutator.NoopMutator{},
	}

	// 3. Create Pipeline Proxy Handler
	pipelineProxy := proxy.NewPipelineProxy(store, authenticators, authorizers, mutators)

	// 3a. Uniform health/readiness/identity contract (ADR 0001).
	healthHandler := health.NewHandler(checker, func() health.Info {
		return health.Info{
			Engine:       "aegis",
			Version:      version.Version,
			BuildSHA:     version.BuildSHA,
			Capabilities: []string{"proxy", "authn", "authz", "mutation"},
			InstanceID:   instanceID,
			StartedAt:    startedAt,
		}
	})

	// 3b. Admin REST API (console rule management)
	adminServer := httpTransport.NewServer(store, healthHandler, pipelineProxy)
	adminHandler := httpx.Chain(adminServer.Routes(),
		httpx.RequestID,
		httpx.Recover(logger),
		metrics.HTTPMiddleware("aegis_admin"),
		httpx.AccessLog(logger),
		httpx.Timeout(c.RequestTimeout),
		httpx.CORS(httpx.CORSConfig{AllowedOrigins: []string{c.CORSOrigins}}),
	)

	adminHTTPServer := &http.Server{
		Addr:         ":" + c.AdminPort,
		Handler:      adminHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	proxyHandler := metrics.HTTPMiddleware("aegis")(pipelineProxy)

	httpServer := &http.Server{
		Addr:         ":" + c.Port,
		Handler:      proxyHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Optional control-plane registration (Argus). No-op unless
	// AUTORIX_ARGUS_URL/AUTORIX_ENROLLMENT_TOKEN are set; never blocks or
	// fails engine startup.
	// aegis has no Postgres of its own (file-based rule store) and no
	// configured nexus/janus upstream address today (internal/authorizer's
	// NexusAuthorizer is still a stub), so it has no honest dependency to
	// declare yet — see platform registry client for the mechanism once
	// aegis gains real upstream addresses.
	registryClient := registry.NewFromEnv("aegis", registry.Endpoints{REST: "http://localhost:" + c.AdminPort},
		[]string{"health.v1"},
		nil,
		logger)
	registryClient.Start(ctx)

	err := run.Run(ctx, c.ShutdownTimeout, logger, []run.Named{
		{
			Name: "aegis-proxy",
			Serve: func() error {
				logger.Info("aegis proxy listening", "port", c.Port)
				if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					return err
				}
				return run.ErrServerClosed
			},
			Shutdown: httpServer.Shutdown,
		},
		{
			Name: "aegis-admin",
			Serve: func() error {
				logger.Info("aegis admin api listening", "port", c.AdminPort)
				if err := adminHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					return err
				}
				return run.ErrServerClosed
			},
			Shutdown: adminHTTPServer.Shutdown,
		},
		{
			Name:     "argus-registry",
			Serve:    func() error { <-ctx.Done(); return run.ErrServerClosed },
			Shutdown: registryClient.Stop,
		},
	})
	if err != nil {
		logger.Error("server error", "error", err)
	}

	logger.Info("autorix aegis stopped")
}
