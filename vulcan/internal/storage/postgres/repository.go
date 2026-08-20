package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/autorix/platform/paging"
	"github.com/autorix/vulcan/internal/core"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("record not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Pool exposes the underlying connection pool so callers (e.g. the readiness
// checker) can probe database reachability without duplicating connection
// wiring.
func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}

// CreateKey inserts a new API key record
func (r *Repository) CreateKey(ctx context.Context, k *core.APIKey) error {
	now := time.Now()
	if k.Scopes == nil {
		k.Scopes = []string{}
	}
	query := `
		INSERT INTO api_keys (
			id, key_prefix, key_hint, key_hash, root_signature_key,
			prev_key_hash, prev_root_signature_key, grace_period_expires_at,
			name, description, owner_id, scopes, expires_at,
			call_count, last_source_ip, state, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $17)
	`

	_, err := r.pool.Exec(ctx, query,
		k.ID, k.KeyPrefix, k.KeyHint, k.KeyHash, k.RootSignatureKey,
		k.PrevKeyHash, k.PrevRootSignatureKey, k.GracePeriodExpiresAt,
		k.Name, k.Description, k.OwnerID, k.Scopes, k.ExpiresAt,
		k.CallCount, k.LastSourceIP, k.State, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert api key: %w", err)
	}

	k.CreatedAt = now
	k.UpdatedAt = now
	return nil
}

const selectKeyColumns = `
	id, key_prefix, key_hint, key_hash, root_signature_key,
	prev_key_hash, prev_root_signature_key, grace_period_expires_at,
	name, description, owner_id, scopes, expires_at, last_used_at,
	call_count, last_source_ip, state, created_at, updated_at
`

func scanKey(row pgx.Row, k *core.APIKey) error {
	var prevKeyHash, prevRootSig *string
	err := row.Scan(
		&k.ID, &k.KeyPrefix, &k.KeyHint, &k.KeyHash, &k.RootSignatureKey,
		&prevKeyHash, &prevRootSig, &k.GracePeriodExpiresAt,
		&k.Name, &k.Description, &k.OwnerID, &k.Scopes, &k.ExpiresAt, &k.LastUsedAt,
		&k.CallCount, &k.LastSourceIP, &k.State, &k.CreatedAt, &k.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if prevKeyHash != nil {
		k.PrevKeyHash = *prevKeyHash
	}
	if prevRootSig != nil {
		k.PrevRootSignatureKey = *prevRootSig
	}
	return nil
}

// GetKeyByID fetches an API key by its UUID
func (r *Repository) GetKeyByID(ctx context.Context, id uuid.UUID) (*core.APIKey, error) {
	query := `SELECT ` + selectKeyColumns + ` FROM api_keys WHERE id = $1`

	var k core.APIKey
	if err := scanKey(r.pool.QueryRow(ctx, query, id), &k); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query api key by id: %w", err)
	}

	return &k, nil
}

// GetKeyByHash finds an API key by its SHA-256 hash, or matching previous hash if within grace period
func (r *Repository) GetKeyByHash(ctx context.Context, keyHash string) (*core.APIKey, error) {
	query := `
		SELECT ` + selectKeyColumns + `
		FROM api_keys
		WHERE state = 'active'
		  AND (
		      key_hash = $1
		      OR (prev_key_hash = $1 AND grace_period_expires_at > CURRENT_TIMESTAMP)
		  )
	`

	var k core.APIKey
	if err := scanKey(r.pool.QueryRow(ctx, query, keyHash), &k); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query api key by hash: %w", err)
	}

	return &k, nil
}

// UpdateKey updates specific fields of an API key (name, description, scopes, expires_at)
func (r *Repository) UpdateKey(ctx context.Context, id uuid.UUID, req core.UpdateKeyRequest) (*core.APIKey, error) {
	existing, err := r.GetKeyByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Scopes != nil {
		existing.Scopes = *req.Scopes
	}
	if req.ExpiresAt != nil {
		existing.ExpiresAt = req.ExpiresAt
	}

	query := `
		UPDATE api_keys
		SET name = $1, description = $2, scopes = $3, expires_at = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
		RETURNING updated_at
	`
	err = r.pool.QueryRow(ctx, query, existing.Name, existing.Description, existing.Scopes, existing.ExpiresAt, id).Scan(&existing.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to update api key: %w", err)
	}

	return existing, nil
}

// RotateKey generates a new key secret and retains the previous secret for a grace period
func (r *Repository) RotateKey(ctx context.Context, id uuid.UUID, newHash, newHint, newRootKey string, gracePeriodExpiresAt *time.Time) (*core.APIKey, error) {
	existing, err := r.GetKeyByID(ctx, id)
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE api_keys
		SET prev_key_hash = key_hash,
		    prev_root_signature_key = root_signature_key,
		    key_hash = $1,
		    key_hint = $2,
		    root_signature_key = $3,
		    grace_period_expires_at = $4,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
		RETURNING updated_at
	`

	err = r.pool.QueryRow(ctx, query, newHash, newHint, newRootKey, gracePeriodExpiresAt, id).Scan(&existing.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to rotate api key: %w", err)
	}

	existing.PrevKeyHash = existing.KeyHash
	existing.PrevRootSignatureKey = existing.RootSignatureKey
	existing.KeyHash = newHash
	existing.KeyHint = newHint
	existing.RootSignatureKey = newRootKey
	existing.GracePeriodExpiresAt = gracePeriodExpiresAt

	return existing, nil
}

// RevokeKey marks an API key as revoked
func (r *Repository) RevokeKey(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_keys SET state = 'revoked', updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RecordUsage updates last_used_at timestamp, increments call_count, and stores source IP
func (r *Repository) RecordUsage(ctx context.Context, id uuid.UUID, sourceIP string) error {
	query := `
		UPDATE api_keys
		SET last_used_at = CURRENT_TIMESTAMP,
		    call_count = call_count + 1,
		    last_source_ip = CASE WHEN $2 != '' THEN $2 ELSE last_source_ip END
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, sourceIP)
	if err != nil {
		return fmt.Errorf("failed to record api key usage: %w", err)
	}
	return nil
}

// UpdateLastUsed updates the last_used_at timestamp and increments call_count
func (r *Repository) UpdateLastUsed(ctx context.Context, id uuid.UUID) {
	_ = r.RecordUsage(ctx, id, "")
}

// ListKeys fetches registered API keys using real SQL keyset pagination
func (r *Repository) ListKeys(ctx context.Context, limit int, cursor string) ([]core.APIKey, bool, error) {
	query := `SELECT ` + selectKeyColumns + ` FROM api_keys`
	args := []interface{}{}

	if cursor != "" {
		createdAt, id, err := decodeKeyCursor(cursor)
		if err != nil {
			return nil, false, fmt.Errorf("invalid cursor: %w", err)
		}
		query += ` WHERE (created_at, id) < ($1, $2)`
		args = append(args, createdAt, id)
	}

	query += fmt.Sprintf(` ORDER BY created_at DESC, id DESC LIMIT $%d`, len(args)+1)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("failed to list keys: %w", err)
	}
	defer rows.Close()

	var keys []core.APIKey
	for rows.Next() {
		var k core.APIKey
		if err := scanKey(rows, &k); err != nil {
			return nil, false, err
		}
		keys = append(keys, k)
	}

	hasMore := len(keys) > limit
	if hasMore {
		keys = keys[:limit]
	}
	return keys, hasMore, nil
}

// ListScopes returns all scopes registered in the catalogue
func (r *Repository) ListScopes(ctx context.Context) ([]core.Scope, error) {
	query := `SELECT name, description, created_at FROM api_key_scopes ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list scopes: %w", err)
	}
	defer rows.Close()

	var scopes []core.Scope
	for rows.Next() {
		var s core.Scope
		if err := rows.Scan(&s.Name, &s.Description, &s.CreatedAt); err != nil {
			return nil, err
		}
		scopes = append(scopes, s)
	}
	if scopes == nil {
		scopes = []core.Scope{}
	}
	return scopes, nil
}

// CreateScope creates a new scope entry in the catalogue
func (r *Repository) CreateScope(ctx context.Context, s *core.Scope) error {
	now := time.Now()
	query := `INSERT INTO api_key_scopes (name, description, created_at) VALUES ($1, $2, $3)`
	_, err := r.pool.Exec(ctx, query, s.Name, s.Description, now)
	if err != nil {
		return fmt.Errorf("failed to create scope: %w", err)
	}
	s.CreatedAt = now
	return nil
}

// DeleteScope deletes a scope from the catalogue by its name
func (r *Repository) DeleteScope(ctx context.Context, name string) error {
	query := `DELETE FROM api_key_scopes WHERE name = $1`
	tag, err := r.pool.Exec(ctx, query, name)
	if err != nil {
		return fmt.Errorf("failed to delete scope: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// decodeKeyCursor reverses the "createdAt|id" cursor payload encoded by
// callers via paging.EncodeCursor.
func decodeKeyCursor(cursor string) (time.Time, uuid.UUID, error) {
	decoded, err := paging.DecodeCursor(cursor)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	parts := strings.SplitN(decoded, "|", 2)
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, errors.New("malformed cursor payload")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor timestamp: %w", err)
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor id: %w", err)
	}
	return createdAt, id, nil
}
