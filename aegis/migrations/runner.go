package migrations

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5"
)

const migrationAdvisoryLock int64 = 0x41454749534d4947 // "AEGISMIG"

var migrationName = regexp.MustCompile(`^([0-9]+)_[a-z0-9][a-z0-9_]*\.up\.sql$`)

// Migration is an immutable, embedded SQL migration.
type Migration struct {
	Version  string
	Name     string
	SQL      string
	Checksum string
}

// Load returns embedded up migrations in ascending numeric version order.
// Duplicate numeric versions and malformed filenames are rejected rather than
// guessed: migration ordering is a security and recovery boundary.
func Load(files embed.FS) ([]Migration, error) {
	return load(files)
}

func load(files fs.FS) ([]Migration, error) {
	migrations := make([]Migration, 0)
	versions := make(map[uint64]struct{})
	err := fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		baseName := path.Base(name)
		matches := migrationName.FindStringSubmatch(baseName)
		if matches == nil {
			return fmt.Errorf("invalid migration filename %q", name)
		}
		version := matches[1]
		numericVersion, err := strconv.ParseUint(version, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid migration version %q: %w", version, err)
		}
		if _, exists := versions[numericVersion]; exists {
			return fmt.Errorf("duplicate migration version %q", version)
		}
		versions[numericVersion] = struct{}{}
		contents, err := fs.ReadFile(files, name)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", name, err)
		}
		sum := sha256.Sum256(contents)
		migrations = append(migrations, Migration{
			Version: version, Name: name, SQL: string(contents), Checksum: hex.EncodeToString(sum[:]),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}

	sort.Slice(migrations, func(i, j int) bool {
		left, _ := strconv.ParseUint(migrations[i].Version, 10, 64)
		right, _ := strconv.ParseUint(migrations[j].Version, 10, 64)
		return left < right
	})
	return migrations, nil
}

// Apply applies all pending embedded migrations in one transaction. It uses a
// transaction-scoped PostgreSQL advisory lock so simultaneous migration jobs
// cannot race. Previously applied migrations are checksum-validated before
// any schema change is attempted.
func Apply(ctx context.Context, conn *pgx.Conn) error {
	if conn == nil {
		return errors.New("migration connection is required")
	}
	migrations, err := Load(Files)
	if err != nil {
		return err
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", migrationAdvisoryLock); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		checksum CHAR(64) NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied := make(map[string]string, len(migrations))
	rows, err := tx.Query(ctx, "SELECT version, checksum FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("read applied migrations: %w", err)
	}
	for rows.Next() {
		var version, checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			rows.Close()
			return fmt.Errorf("scan applied migration: %w", err)
		}
		applied[version] = checksum
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate applied migrations: %w", err)
	}
	rows.Close()

	for _, migration := range migrations {
		if checksum, exists := applied[migration.Version]; exists {
			if checksum != migration.Checksum {
				return fmt.Errorf("migration %s checksum mismatch", migration.Name)
			}
			continue
		}
		if _, err := tx.Exec(ctx, migration.SQL); err != nil {
			return fmt.Errorf("apply migration %s: %w", migration.Name, err)
		}
		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version, checksum) VALUES ($1, $2)",
			migration.Version, migration.Checksum); err != nil {
			return fmt.Errorf("record migration %s: %w", migration.Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	return nil
}
