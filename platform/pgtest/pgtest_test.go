package pgtest_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/autorix/platform/pgtest"
)

func TestStartPostgres_AppliesMigrationsAndReturnsAUsablePool(t *testing.T) {
	dir := t.TempDir()
	migration := `CREATE TABLE widgets (id SERIAL PRIMARY KEY, name TEXT NOT NULL);`
	if err := os.WriteFile(filepath.Join(dir, "000001_create_widgets.up.sql"), []byte(migration), 0o600); err != nil {
		t.Fatalf("writing fixture migration: %v", err)
	}

	pool := pgtest.StartPostgres(t, dir)

	if _, err := pool.Exec(context.Background(), `INSERT INTO widgets (name) VALUES ('gizmo')`); err != nil {
		t.Fatalf("expected insert against migrated schema to succeed, got: %v", err)
	}

	var count int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM widgets`).Scan(&count); err != nil {
		t.Fatalf("unexpected error querying: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row, got %d", count)
	}
}

// TestStartPostgres_ReturnedPoolUsesExtendedProtocolForJSONBParams proves
// the returned pool does NOT inherit the simple-protocol mode migrations
// need internally. Under simple protocol, pgx encodes a []byte parameter
// as an untyped bytea literal, which fails against a jsonb column with
// "invalid input syntax for type json" — exactly what every engine's
// repository does when inserting marshaled JSON traits/context.
func TestStartPostgres_ReturnedPoolUsesExtendedProtocolForJSONBParams(t *testing.T) {
	dir := t.TempDir()
	migration := `CREATE TABLE jsonb_test (id SERIAL PRIMARY KEY, payload JSONB NOT NULL);`
	if err := os.WriteFile(filepath.Join(dir, "000001_create.up.sql"), []byte(migration), 0o600); err != nil {
		t.Fatalf("writing fixture migration: %v", err)
	}

	pool := pgtest.StartPostgres(t, dir)

	payload := []byte(`{"email":"elena@autorix.io"}`)
	if _, err := pool.Exec(context.Background(), `INSERT INTO jsonb_test (payload) VALUES ($1)`, payload); err != nil {
		t.Fatalf("expected a []byte param to insert cleanly into jsonb, got: %v", err)
	}
}

func TestStartPostgres_AppliesMultipleMigrationsInOrder(t *testing.T) {
	dir := t.TempDir()
	first := `CREATE TABLE ordered_test (id SERIAL PRIMARY KEY);`
	second := `ALTER TABLE ordered_test ADD COLUMN label TEXT;`
	if err := os.WriteFile(filepath.Join(dir, "000001_create.up.sql"), []byte(first), 0o600); err != nil {
		t.Fatalf("writing fixture migration: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "000002_alter.up.sql"), []byte(second), 0o600); err != nil {
		t.Fatalf("writing fixture migration: %v", err)
	}

	pool := pgtest.StartPostgres(t, dir)

	if _, err := pool.Exec(context.Background(), `INSERT INTO ordered_test (label) VALUES ('ok')`); err != nil {
		t.Fatalf("expected the second migration's column to exist, got: %v", err)
	}
}
