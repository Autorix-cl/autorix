// Package pgtest spins up a real Postgres instance via testcontainers-go
// for repository-layer integration tests (P1-S5-T3). Every engine's
// internal/storage/postgres package had zero tests before this — unit
// tests alone can't catch a SQL error, a migration that doesn't match the
// repository's assumptions, or a constraint violation the code doesn't
// handle.
package pgtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// StartPostgres starts a fresh, isolated Postgres 16 container, applies
// every "*.up.sql" file in migrationsDir (sorted lexically — the same
// ordering convention the migration filenames themselves already use), and
// returns a pool connected to it. The container and pool are torn down
// automatically via t.Cleanup, and the test is skipped in -short mode so a
// fast local `go test -short ./...` doesn't need Docker.
func StartPostgres(t *testing.T, migrationsDir string) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("pgtest: skipping testcontainers-backed integration test in -short mode")
	}

	ctx := context.Background()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("autorix_test"),
		tcpostgres.WithUsername("autorix"),
		tcpostgres.WithPassword("autorix_test_password"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
	)
	if err != nil {
		t.Fatalf("pgtest: starting postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("pgtest: terminating container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("pgtest: getting connection string: %v", err)
	}

	// Migrations are applied over a dedicated connection forced into the
	// simple protocol, since a migration file may contain multiple
	// statements. That mode must NOT leak into the pool returned to the
	// caller: under simple protocol pgx encodes a []byte parameter as an
	// untyped bytea literal, which fails against a jsonb column with
	// "invalid input syntax for type json" — exactly what every engine's
	// repository does when inserting marshaled JSON. So migrations run
	// over their own connection, closed before the real pool is built.
	if err := applyMigrations(ctx, dsn, migrationsDir); err != nil {
		t.Fatalf("pgtest: applying migrations from %s: %v", migrationsDir, err)
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("pgtest: parsing DSN: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("pgtest: creating pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

func applyMigrations(ctx context.Context, dsn, dir string) error {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("parsing DSN: %w", err)
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	var conn *pgx.Conn
	for attempt := 0; attempt < 20; attempt++ {
		conn, err = pgx.ConnectConfig(ctx, cfg)
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	if err != nil {
		return fmt.Errorf("connecting: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		sqlBytes, err := os.ReadFile(filepath.Join(dir, name)) // #nosec G304 -- dir is a test-supplied constant, never external input
		if err != nil {
			return fmt.Errorf("reading %s: %w", name, err)
		}
		if _, err := conn.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("applying %s: %w", name, err)
		}
	}
	return nil
}
