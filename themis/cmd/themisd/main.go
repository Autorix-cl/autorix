package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	pb "github.com/autorix/themis/api/autorix/themis/v1"
	"github.com/autorix/themis/internal/core"
	"github.com/autorix/themis/internal/engine"
	"github.com/autorix/themis/internal/storage/postgres"
	grpchandler "github.com/autorix/themis/internal/transport/grpc"
	httphandler "github.com/autorix/themis/internal/transport/http"

	"github.com/autorix/platform/config"
	"github.com/autorix/platform/grpchealth"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/httpx"
	platformlog "github.com/autorix/platform/log"
	"github.com/autorix/platform/metrics"
	platformpg "github.com/autorix/platform/postgres"
	"github.com/autorix/platform/registry"
	"github.com/autorix/platform/run"
	"github.com/autorix/platform/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// cfg is themis's environment configuration, loaded via platform/config.
// Every field keeps the exact default it had before this refactor.
type cfg struct {
	DatabaseURL     string        `env:"DATABASE_URL" envDefault:"postgres://postgres:postgres@localhost:5432/autorix_themis?sslmode=disable"`
	HTTPPort        string        `env:"HTTP_PORT" envDefault:"4488"`
	Port            string        `env:"PORT" envDefault:"50052"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info"`
	InstanceID      string        `env:"AUTORIX_INSTANCE_ID"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	RequestTimeout  time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
	CORSOrigins     string        `env:"CORS_ORIGINS" envDefault:"*"`
}

func main() {
	var c cfg
	if err := config.Load(&c); err != nil {
		panic("Failed to load config: " + err.Error())
	}

	startedAt := time.Now()

	instanceID := c.InstanceID
	if instanceID == "" {
		if hostname, hostErr := os.Hostname(); hostErr == nil {
			instanceID = hostname
		}
	}

	logger := platformlog.New(platformlog.Config{
		Engine:     "themis",
		InstanceID: instanceID,
		Level:      c.LogLevel,
	}, nil)

	logger.Info("starting autorix themis (business rules engine)")

	ctx := context.Background()

	// 1. Initialize DB Connection from environment (12-Factor App). Connect
	// retries the boot-time Ping with backoff instead of dying on the first
	// failure — themis previously used pgxpool.New directly, which had no
	// retry at all.
	pool, err := platformpg.Connect(ctx, c.DatabaseURL, platformpg.ConnectOptions{})
	if err != nil {
		logger.Error("unable to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	metrics.RegisterPostgresPool(pool, "themis")

	// 2. Initialize Infrastructure Layer (Repository)
	repo := postgres.NewRepository(pool)

	// 3. Initialize Domain Layer (CEL Evaluator + Service)
	celEvaluator, err := engine.NewCELEvaluator()
	if err != nil {
		logger.Error("failed to initialize CEL evaluator", "error", err)
		os.Exit(1)
	}

	service := core.NewService(repo, celEvaluator)

	// 3b. Uniform health/readiness/identity contract (ADR 0001). Readiness
	// checks Postgres reachability, bounded independently of the caller's
	// context.
	checker := health.NewChecker()
	checker.Register("postgres", platformpg.Check(repo.Pool()))

	healthHandler := health.NewHandler(checker, func() health.Info {
		return health.Info{
			Engine:       "themis",
			Version:      version.Version,
			BuildSHA:     version.BuildSHA,
			Capabilities: []string{"abac", "cel"},
			InstanceID:   instanceID,
			StartedAt:    startedAt,
		}
	})

	// 4. Initialize Transport Layer (REST admin API). The shared httpx
	// middleware chain wraps only this REST server — the gRPC server below
	// is HTTP-only in transport, not an httpx.Handler.
	httpServerHandler := httphandler.NewServer(service, healthHandler)
	restHandler := httpx.Chain(httpServerHandler.Routes(),
		httpx.RequestID,
		httpx.Recover(logger),
		metrics.HTTPMiddleware("themis"),
		httpx.AccessLog(logger),
		httpx.Timeout(c.RequestTimeout),
		httpx.CORS(httpx.CORSConfig{AllowedOrigins: []string{c.CORSOrigins}}),
	)

	httpServer := &http.Server{
		Addr:    ":" + c.HTTPPort,
		Handler: restHandler,
		// Bounds how long the server waits to read request headers, so a
		// client that trickles bytes in (Slowloris) can't tie up a
		// connection indefinitely.
		ReadHeaderTimeout: 10 * time.Second,
	}

	// 5. Initialize Transport Layer (gRPC Server)
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(metrics.UnaryServerInterceptor("themis")),
		grpc.StreamInterceptor(metrics.StreamServerInterceptor("themis")),
	)
	themisServer := grpchandler.NewServer(service)

	pb.RegisterThemisServiceServer(grpcServer, themisServer)

	// Register the standard grpc.health.v1.Health service over the same
	// checker HTTP readiness uses, so a gRPC-native probe and the HTTP probe
	// can never disagree.
	grpchealth.Register(grpcServer, checker, "autorix.themis.v1.ThemisService")

	// Enable reflection for tools like grpcurl
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":"+c.Port)
	if err != nil {
		logger.Error("failed to listen", "port", c.Port, "error", err)
		os.Exit(1)
	}

	// 6. Graceful shutdown of both servers — added for the first time here;
	// themis previously ran grpcServer.Serve blocking with no signal handling
	// at all, so a SIGTERM killed the process mid-request. This is an
	// intentional behavior improvement, not a like-for-like refactor.
	runCtx, stop := run.NotifyContext()
	defer stop()

	// Optional control-plane registration (Argus). No-op unless
	// AUTORIX_ARGUS_URL/AUTORIX_ENROLLMENT_TOKEN are set; never blocks or
	// fails engine startup.
	registryClient := registry.NewFromEnv("themis", registry.Endpoints{
		REST: "http://localhost:" + c.HTTPPort,
		GRPC: "localhost:" + c.Port,
	},
		[]string{"health.v1", "grpc-health.v1"},
		[]registry.Dependency{{Name: "postgres", Target: registry.PostgresDependencyTarget(c.DatabaseURL)}},
		logger)
	registryClient.Start(runCtx)

	err = run.Run(runCtx, c.ShutdownTimeout, logger, []run.Named{
		{
			Name: "themis-http",
			Serve: func() error {
				logger.Info("themis rest api listening", "port", c.HTTPPort)
				if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					return err
				}
				return run.ErrServerClosed
			},
			Shutdown: httpServer.Shutdown,
		},
		{
			Name: "themis-grpc",
			Serve: func() error {
				logger.Info("themis listening", "port", c.Port, "transport", "gRPC")
				if err := grpcServer.Serve(lis); err != nil {
					return err
				}
				return run.ErrServerClosed
			},
			// GracefulStop takes no context and returns no error; the closure
			// still respects run.Named.Shutdown's signature. If GracefulStop
			// outlives ctx's drain deadline, Run's own timeout logs a warning
			// but does not force-kill the gRPC server — a known, accepted
			// limitation for this iteration.
			Shutdown: func(ctx context.Context) error {
				grpcServer.GracefulStop()
				return nil
			},
		},
		{
			Name:     "argus-registry",
			Serve:    func() error { <-runCtx.Done(); return run.ErrServerClosed },
			Shutdown: registryClient.Stop,
		},
	})
	if err != nil {
		logger.Error("server error", "error", err)
	}

	logger.Info("autorix themis stopped")
}
