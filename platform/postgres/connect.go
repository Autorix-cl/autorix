// This file extends the postgres package with pool construction, boot-time
// connection retry and a schema-migration readiness check (P1-S2-T5).
// Today a failed DSN kills an engine's process immediately with no retry —
// Connect replaces that with bounded, backed-off retries so a database that
// is merely slow to come up (a common ordering race in docker compose and
// Kubernetes alike) does not crash the engine that depends on it.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectOptions configures pool limits and boot-time retry behavior.
// A zero value uses sane defaults for all four fields.
type ConnectOptions struct {
	// MaxConns caps the pool size. Defaults to 10.
	MaxConns int32
	// MinConns is the pool's minimum idle size. Defaults to 0.
	MinConns int32
	// MaxRetries bounds how many times Connect pings before giving up.
	// Defaults to 5.
	MaxRetries int
	// RetryBackoff is the delay between retry attempts. Defaults to 2s.
	RetryBackoff time.Duration
}

func (o ConnectOptions) withDefaults() ConnectOptions {
	if o.MaxConns <= 0 {
		o.MaxConns = 10
	}
	if o.MaxRetries <= 0 {
		o.MaxRetries = 5
	}
	if o.RetryBackoff <= 0 {
		o.RetryBackoff = 2 * time.Second
	}
	return o
}

// BuildPoolConfig parses dsn and applies opts' pool limits, without opening
// any connection. Exposed separately from Connect so the limit-application
// logic is unit-testable without a live database.
func BuildPoolConfig(dsn string, opts ConnectOptions) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: invalid DSN: %w", err)
	}
	opts = opts.withDefaults()
	cfg.MaxConns = opts.MaxConns
	cfg.MinConns = opts.MinConns
	return cfg, nil
}

// RetryPing calls ping up to maxAttempts times, waiting backoff between
// attempts, returning nil on the first success. It returns ctx.Err() if ctx
// is cancelled while waiting between attempts, or the last error once
// maxAttempts is exhausted. Exported so Connect's retry behavior is
// directly unit-testable without a real database.
func RetryPing(ctx context.Context, maxAttempts int, backoff time.Duration, ping func(ctx context.Context) error) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = ping(ctx)
		if lastErr == nil {
			return nil
		}
		if attempt == maxAttempts {
			break
		}
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return lastErr
}

// Connect builds a pool from dsn with opts' limits, then retries Ping up to
// opts.MaxRetries times before giving up. On failure the pool is closed and
// no reference is leaked to the caller.
func Connect(ctx context.Context, dsn string, opts ConnectOptions) (*pgxpool.Pool, error) {
	cfg, err := BuildPoolConfig(dsn, opts)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: creating pool: %w", err)
	}

	opts = opts.withDefaults()
	if err := RetryPing(ctx, opts.MaxRetries, opts.RetryBackoff, pool.Ping); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: could not reach database after %d attempts: %w", opts.MaxRetries, err)
	}

	return pool, nil
}
