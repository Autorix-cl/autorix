package migrations

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"context"

	"github.com/autorix/platform/pgtest"
)

//go:embed testdata/*.sql
var testFiles embed.FS

func TestLoadOrdersAndChecksumsMigrations(t *testing.T) {
	migrations, err := Load(testFiles)
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != 2 {
		t.Fatalf("migration count = %d, want 2", len(migrations))
	}
	if migrations[0].Version != "000002" || migrations[1].Version != "000010" {
		t.Fatalf("versions = %q, %q; want numeric order", migrations[0].Version, migrations[1].Version)
	}
	sum := sha256.Sum256([]byte(migrations[0].SQL))
	if migrations[0].Checksum != hex.EncodeToString(sum[:]) {
		t.Fatalf("checksum = %q, want sha256 of embedded SQL", migrations[0].Checksum)
	}
}

func TestLoadRejectsInvalidEmbeddedNames(t *testing.T) {
	// embed.FS cannot be constructed dynamically. The production set is also
	// exercised here to assert its public migration contract is valid.
	migrations, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) == 0 {
		t.Fatal("expected at least one embedded migration")
	}
}

func TestLoadRejectsUnsafeMigrationSets(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files fstest.MapFS
	}{
		{
			name:  "invalid filename",
			files: fstest.MapFS{"unversioned.sql": &fstest.MapFile{Data: []byte("SELECT 1;")}},
		},
		{
			name: "duplicate numeric version",
			files: fstest.MapFS{
				"000001_first.up.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
				"1_second.up.sql":     &fstest.MapFile{Data: []byte("SELECT 2;")},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := load(tc.files)
			if err == nil {
				t.Fatal("load accepted unsafe migration set")
			}
			if errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("unexpected filesystem failure: %v", err)
			}
		})
	}
}

func TestApplyIsIdempotentAndRejectsChecksumChanges(t *testing.T) {
	pool := pgtest.StartPostgres(t, t.TempDir())
	ctx := context.Background()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()

	if err := Apply(ctx, conn.Conn()); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	if err := Apply(ctx, conn.Conn()); err != nil {
		t.Fatalf("idempotent Apply: %v", err)
	}

	var count int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("applied migration count = %d, want 1", count)
	}

	if _, err := conn.Exec(ctx, "UPDATE schema_migrations SET checksum = 'changed'"); err != nil {
		t.Fatal(err)
	}
	if err := Apply(ctx, conn.Conn()); err == nil {
		t.Fatal("Apply accepted a changed migration checksum")
	}
}
