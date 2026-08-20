package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/autorix/platform/postgres"
)

func TestSchemaVersionCheck_PassesWhenAppliedCountMatchesExpected(t *testing.T) {
	q := &stubQueryer{count: 7}
	check := postgres.SchemaVersionCheck(q, "schema_migrations", 7)

	if err := check(context.Background()); err != nil {
		t.Fatalf("expected nil error when counts match, got %v", err)
	}
}

func TestSchemaVersionCheck_FailsWhenAppliedCountDoesNotMatch(t *testing.T) {
	q := &stubQueryer{count: 4}
	check := postgres.SchemaVersionCheck(q, "schema_migrations", 7)

	if err := check(context.Background()); err == nil {
		t.Fatalf("expected an error when applied count (4) does not match expected (7)")
	}
}

func TestSchemaVersionCheck_FailsWhenQueryErrors(t *testing.T) {
	q := &stubQueryer{queryErr: errors.New("relation does not exist")}
	check := postgres.SchemaVersionCheck(q, "schema_migrations", 7)

	if err := check(context.Background()); err == nil {
		t.Fatalf("expected an error when the migrations table query fails")
	}
}

// stubQueryer implements postgres.Queryer directly for these tests.
type stubQueryer struct {
	count    int
	queryErr error
}

func (s *stubQueryer) QueryRow(ctx context.Context, sql string, args ...interface{}) postgres.Row {
	return stubRow{count: s.count, err: s.queryErr}
}

type stubRow struct {
	count int
	err   error
}

func (r stubRow) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	p, ok := dest[0].(*int)
	if !ok {
		return errors.New("unexpected scan target")
	}
	*p = r.count
	return nil
}
