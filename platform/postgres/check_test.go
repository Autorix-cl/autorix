package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/autorix/platform/postgres"
)

type fakePinger struct {
	err   error
	delay time.Duration
}

func (f fakePinger) Ping(ctx context.Context) error {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return f.err
}

func TestCheck_ReturnsNilWhenPingSucceeds(t *testing.T) {
	fn := postgres.Check(fakePinger{})

	if err := fn(context.Background()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestCheck_ReturnsErrorWhenPingFails(t *testing.T) {
	fn := postgres.Check(fakePinger{err: errors.New("connection refused")})

	if err := fn(context.Background()); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCheck_BoundsPingWithATimeout(t *testing.T) {
	fn := postgres.Check(fakePinger{delay: 5 * time.Second})

	start := time.Now()
	err := fn(context.Background())
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if elapsed > 3*time.Second {
		t.Fatalf("expected Check to bound the ping with its own timeout, took %v", elapsed)
	}
}
