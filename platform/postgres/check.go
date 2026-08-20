// Package postgres provides a readiness Check helper for engines backed by
// pgxpool, bounded so a stalled database can never stall an engine's own
// readiness endpoint indefinitely (P1-S1-T2).
package postgres

import (
	"context"
	"time"

	"github.com/autorix/platform/health"
)

// pingTimeout bounds every readiness ping independently of the caller's
// context, so a slow database degrades readiness within a fixed budget
// instead of hanging on whatever deadline the HTTP handler happened to set.
const pingTimeout = 2 * time.Second

// Pinger is satisfied by *pgxpool.Pool. Accepting the interface rather than
// the concrete type keeps this package testable without a live database.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Check wraps pool.Ping into a health.CheckFunc bounded by pingTimeout.
func Check(pool Pinger) health.CheckFunc {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, pingTimeout)
		defer cancel()
		return pool.Ping(ctx)
	}
}
