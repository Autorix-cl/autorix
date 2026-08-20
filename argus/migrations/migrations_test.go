package migrations_test

import (
	"context"
	"testing"

	"github.com/autorix/platform/pgtest"
)

// TestMigrationsApplyCleanly is the fastest possible check that the SQL in
// this directory is valid and self-consistent — pgtest.StartPostgres
// applies every *.up.sql file against a real Postgres 16 container and
// fails loudly if any of them errors.
func TestMigrationsApplyCleanly(t *testing.T) {
	pool := pgtest.StartPostgres(t, ".")

	var count int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM engine_types`).Scan(&count); err != nil {
		t.Fatalf("querying seeded engine_types: %v", err)
	}
	if count != 7 {
		t.Fatalf("expected 7 seeded engine types, got %d", count)
	}
}
