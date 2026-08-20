package main

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"os"
	"time"

	argusv1 "github.com/autorix/argus/api/autorix/argus/v1"
	"github.com/autorix/argus/internal/core"
	"github.com/autorix/argus/internal/credential"
	"github.com/autorix/argus/internal/storage/postgres"
	grpchandler "github.com/autorix/argus/internal/transport/grpc"
	httphandler "github.com/autorix/argus/internal/transport/http"

	"github.com/autorix/platform/config"
	"github.com/autorix/platform/grpchealth"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/httpx"
	platformlog "github.com/autorix/platform/log"
	platformpg "github.com/autorix/platform/postgres"
	"github.com/autorix/platform/run"
	"github.com/autorix/platform/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// cfg is argus's environment configuration, loaded via platform/config —
// the same pattern every other engine's main.go uses (P1-S2).
type cfg struct {
	DatabaseURL     string        `env:"DATABASE_URL" envDefault:"postgres://postgres:postgres@localhost:5432/autorix_argus?sslmode=disable"`
	HTTPPort        string        `env:"HTTP_PORT" envDefault:"4400"`
	Port            string        `env:"PORT" envDefault:"50053"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info"`
	InstanceID      string        `env:"AUTORIX_INSTANCE_ID"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	RequestTimeout  time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
	CORSOrigins     string        `env:"CORS_ORIGINS" envDefault:"*"`

	// CredentialEncKeyHex is a 32-byte AES-256-GCM key (64 hex chars)
	// sealing instance credentials at rest so HMAC heartbeat signatures
	// (P2-S3-T4) can be verified. Left unset in dev, an ephemeral key is
	// generated at boot — fine for a single-process dev run, but any
	// restart invalidates every previously-issued instance credential, so
	// production deployments must set this explicitly.
	CredentialEncKeyHex string `env:"ARGUS_CREDENTIAL_ENC_KEY" envDefault:""`

	// EvaluatorSweepInterval is how often the background lifecycle sweep
	// (P2-S5-T2) advances stale instances that nobody is actively
	// heartbeating from or querying right now.
	EvaluatorSweepInterval time.Duration `env:"ARGUS_EVALUATOR_SWEEP_INTERVAL" envDefault:"30s"`

	// RetentionSweepInterval and RetentionAfter drive the periodic prune
	// (P2-S2-T6) of old timeline events and long-evicted instances.
	RetentionSweepInterval time.Duration `env:"ARGUS_RETENTION_SWEEP_INTERVAL" envDefault:"1h"`
	RetentionAfter         time.Duration `env:"ARGUS_RETENTION_AFTER" envDefault:"720h"` // 30 days
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
		Engine:     "argus",
		InstanceID: instanceID,
		Level:      c.LogLevel,
	}, nil)

	logger.Info("starting autorix argus (control plane / fleet registry)")

	ctx := context.Background()

	// 1. Database connection, with boot-time retry (P1-S2).
	pool, err := platformpg.Connect(ctx, c.DatabaseURL, platformpg.ConnectOptions{})
	if err != nil {
		logger.Error("unable to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// 2. Uniform health/readiness/identity contract (ADR 0001). The
	// repository, and the domain service that will back the RPCs the gRPC
	// server currently stubs as Unimplemented, land with P2-S2/P2-S3 —
	// this wiring exists now so Argus is itself a real, honestly-reporting
	// member of the fleet from day one, per the roadmap's "never fake a
	// signal" principle: an Argus that lies about its own health would be
	// a strange thing to build a fleet health system on top of.
	checker := health.NewChecker()
	checker.Register("postgres", platformpg.Check(pool))

	healthHandler := health.NewHandler(checker, func() health.Info {
		return health.Info{
			Engine:     "argus",
			Version:    version.Version,
			BuildSHA:   version.BuildSHA,
			InstanceID: instanceID,
			StartedAt:  startedAt,
		}
	})

	// 3. REST admin API (enrollment tokens, fleet queries — P2-S3/P2-S6).
	repo := postgres.NewRepository(pool)
	httpServerHandler := httphandler.NewServer(healthHandler, repo)
	restHandler := httpx.Chain(httpServerHandler.Routes(),
		httpx.RequestID,
		httpx.Recover(logger),
		httpx.AccessLog(logger),
		httpx.Timeout(c.RequestTimeout),
		httpx.CORS(httpx.CORSConfig{AllowedOrigins: []string{c.CORSOrigins}}),
	)

	httpServer := &http.Server{
		Addr:              ":" + c.HTTPPort,
		Handler:           restHandler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// 4. gRPC hot path (engine registration/heartbeat — P2-S3).
	encKey, err := resolveCredentialEncKey(c.CredentialEncKeyHex, logger)
	if err != nil {
		logger.Error("invalid ARGUS_CREDENTIAL_ENC_KEY", "error", err)
		os.Exit(1)
	}
	grpcServer := grpc.NewServer()
	argusServer := grpchandler.NewServer(repo, encKey)
	argusv1.RegisterArgusServiceServer(grpcServer, argusServer)

	grpchealth.Register(grpcServer, checker, "autorix.argus.v1.ArgusService")
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":"+c.Port)
	if err != nil {
		logger.Error("failed to listen", "port", c.Port, "error", err)
		os.Exit(1)
	}

	runCtx, stop := run.NotifyContext()
	defer stop()

	// 5. First-run bootstrap check (P3-S1-T2): when no operators exist,
	// generate and log a one-time bootstrap token (abt_...) for /setup.
	initBootstrapToken(runCtx, logger, repo)

	// 6. Background lifecycle sweep (P2-S5-T2) and retention prune
	// (P2-S2-T6): both are best-effort maintenance loops, not part of the
	// request-serving contract, so they run detached from run.Run's
	// graceful-shutdown bookkeeping and simply stop when runCtx is
	// cancelled.
	startEvaluatorLoop(runCtx, logger, repo, c.EvaluatorSweepInterval)
	startRetentionLoop(runCtx, logger, repo, c.RetentionSweepInterval, c.RetentionAfter)

	err = run.Run(runCtx, c.ShutdownTimeout, logger, []run.Named{
		{
			Name: "argus-http",
			Serve: func() error {
				logger.Info("argus rest api listening", "port", c.HTTPPort)
				if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					return err
				}
				return run.ErrServerClosed
			},
			Shutdown: httpServer.Shutdown,
		},
		{
			Name: "argus-grpc",
			Serve: func() error {
				logger.Info("argus listening", "port", c.Port, "transport", "gRPC")
				if err := grpcServer.Serve(lis); err != nil {
					return err
				}
				return run.ErrServerClosed
			},
			Shutdown: func(ctx context.Context) error {
				grpcServer.GracefulStop()
				return nil
			},
		},
	})
	if err != nil {
		logger.Error("server error", "error", err)
	}

	logger.Info("autorix argus stopped")
}

// errInvalidEncKeyLength is returned when ARGUS_CREDENTIAL_ENC_KEY is set
// but does not decode to exactly credential.KeySize bytes.
var errInvalidEncKeyLength = errors.New("ARGUS_CREDENTIAL_ENC_KEY must decode to 32 bytes (64 hex characters)")

// resolveCredentialEncKey decodes a 64-hex-char (32-byte) AES-256-GCM key
// from hexKey. An empty hexKey generates and returns an ephemeral one,
// logging a warning — acceptable for local dev, wrong for production,
// where every process restart would otherwise invalidate every
// already-issued instance credential.
func resolveCredentialEncKey(hexKey string, logger interface {
	Warn(msg string, args ...any)
}) ([]byte, error) {
	if hexKey == "" {
		key := make([]byte, credential.KeySize)
		if _, err := cryptorand.Read(key); err != nil {
			return nil, err
		}
		logger.Warn("ARGUS_CREDENTIAL_ENC_KEY not set — generated an ephemeral key; instance credentials will not survive a restart in this configuration")
		return key, nil
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}
	if len(key) != credential.KeySize {
		return nil, errInvalidEncKeyLength
	}
	return key, nil
}

// startEvaluatorLoop runs core.Evaluator.Sweep every interval until ctx is
// cancelled (P2-S5-T2). A failed sweep is logged, not fatal — the next
// tick tries again, the same "never let background maintenance take down
// the process" posture as the retention loop below.
func startEvaluatorLoop(ctx context.Context, logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}, repo core.Repository, interval time.Duration) {
	if interval <= 0 {
		return
	}
	evaluator := core.NewEvaluator(repo)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := evaluator.Sweep(ctx)
				if err != nil {
					logger.Error("evaluator sweep failed", "error", err)
					continue
				}
				if n > 0 {
					logger.Info("evaluator sweep transitioned stale instances", "count", n)
				}
			}
		}
	}()
}

// startRetentionLoop runs Repository.PruneOlderThan every interval until
// ctx is cancelled (P2-S2-T6).
func startRetentionLoop(ctx context.Context, logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}, repo core.Repository, interval time.Duration, after time.Duration) {
	if interval <= 0 || after <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().UTC().Add(-after)
				events, instances, err := repo.PruneOlderThan(ctx, cutoff)
				if err != nil {
					logger.Error("retention prune failed", "error", err)
					continue
				}
				if events > 0 || instances > 0 {
					logger.Info("retention prune completed", "events_deleted", events, "instances_deleted", instances)
				}
			}
		}
	}()
}

func initBootstrapToken(ctx context.Context, logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}, repo core.Repository) {
	count, err := repo.CountOperators(ctx)
	if err != nil {
		logger.Error("failed to check operator count", "error", err)
		return
	}
	if count > 0 {
		return
	}

	hasToken, err := repo.HasValidBootstrapToken(ctx)
	if err != nil {
		logger.Error("failed to check bootstrap token", "error", err)
		return
	}
	if hasToken {
		return
	}

	rawToken, tokenHash, err := credential.GenerateSecret("abt_")
	if err != nil {
		logger.Error("failed to generate bootstrap token", "error", err)
		return
	}

	if err := repo.CreateBootstrapToken(ctx, tokenHash); err != nil {
		logger.Error("failed to persist bootstrap token", "error", err)
		return
	}

	logger.Info("FIRST-RUN BOOTSTRAP TOKEN (Use to create initial Owner at /setup): "+rawToken, "token", rawToken)
}
