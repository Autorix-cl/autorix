package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/autorix/platform/health"
)

// Row is an alias for pgx.Row, kept as its own name in this package so
// callers and tests don't need to import pgx directly for this one type.
type Row = pgx.Row

// Queryer is satisfied by *pgxpool.Pool. Accepting the interface — rather
// than the concrete pool type — keeps SchemaVersionCheck testable without a
// live database, the same reasoning as Pinger in check.go.
type Queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) Row
}

// SchemaVersionCheck returns a health.CheckFunc that fails readiness unless
// table reports exactly expected applied rows. It is opt-in: an engine only
// gains this check once it tracks applied migrations in Postgres via table
// (e.g. a `schema_migrations` table populated by its migration tooling) —
// no Autorix engine does so as of P1-S2, so no engine registers this check
// yet. It exists so an engine can adopt migration-version readiness the
// moment it adopts migration tracking, without inventing the plumbing again.
func SchemaVersionCheck(db Queryer, table string, expected int) health.CheckFunc {
	return func(ctx context.Context) error {
		var count int
		sql := fmt.Sprintf("SELECT COUNT(*) FROM %s", table) // #nosec G201 -- table is a caller-supplied constant, never user input
		if err := db.QueryRow(ctx, sql).Scan(&count); err != nil {
			return fmt.Errorf("checking %s: %w", table, err)
		}
		if count != expected {
			return fmt.Errorf("%s reports %d applied migrations, expected %d", table, count, expected)
		}
		return nil
	}
}
