package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/autorix/platform/postgres"
)

func TestRetryPing_SucceedsOnFirstTry(t *testing.T) {
	calls := 0
	err := postgres.RetryPing(context.Background(), 5, time.Millisecond, func(ctx context.Context) error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call on immediate success, got %d", calls)
	}
}

func TestRetryPing_SucceedsAfterTransientFailures(t *testing.T) {
	calls := 0
	err := postgres.RetryPing(context.Background(), 5, time.Millisecond, func(ctx context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("connection refused")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected exactly 3 calls, got %d", calls)
	}
}

func TestRetryPing_ReturnsLastErrorAfterExhaustingAttempts(t *testing.T) {
	calls := 0
	wantErr := errors.New("still down")
	err := postgres.RetryPing(context.Background(), 3, time.Millisecond, func(ctx context.Context) error {
		calls++
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected the last error to be returned, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected exactly 3 attempts (maxAttempts), got %d", calls)
	}
}

func TestRetryPing_StopsEarlyWhenContextCancelledDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	go func() {
		time.Sleep(15 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := postgres.RetryPing(ctx, 100, 500*time.Millisecond, func(ctx context.Context) error {
		calls++
		return errors.New("down")
	})
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("expected RetryPing to stop promptly on cancellation, took %v", elapsed)
	}
	if calls >= 100 {
		t.Fatalf("expected cancellation to cut retries short, got %d calls", calls)
	}
}

func TestConnect_RejectsAnInvalidDSN(t *testing.T) {
	_, err := postgres.Connect(context.Background(), "not-a-valid-dsn::://", postgres.ConnectOptions{})
	if err == nil {
		t.Fatalf("expected an error for an invalid DSN")
	}
}

func TestConnect_AppliesPoolLimitsFromOptions(t *testing.T) {
	cfg, err := postgres.BuildPoolConfig("postgres://user:pass@localhost:5432/db", postgres.ConnectOptions{
		MaxConns: 25,
		MinConns: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxConns != 25 {
		t.Fatalf("expected MaxConns=25, got %d", cfg.MaxConns)
	}
	if cfg.MinConns != 5 {
		t.Fatalf("expected MinConns=5, got %d", cfg.MinConns)
	}
}

func TestConnect_UsesSaneDefaultsWhenOptionsAreZero(t *testing.T) {
	cfg, err := postgres.BuildPoolConfig("postgres://user:pass@localhost:5432/db", postgres.ConnectOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxConns <= 0 {
		t.Fatalf("expected a positive default MaxConns, got %d", cfg.MaxConns)
	}
	if cfg.MinConns < 0 {
		t.Fatalf("expected a non-negative default MinConns, got %d", cfg.MinConns)
	}
}
