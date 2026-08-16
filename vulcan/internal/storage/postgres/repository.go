package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

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

// CreateKey inserts a new API key record
func (r *Repository) CreateKey(ctx context.Context, k *core.APIKey) error {
	now := time.Now()
	query := `
		INSERT INTO api_keys (
			id, key_prefix, key_hint, key_hash, root_signature_key,
			name, owner_id, scopes, expires_at, state, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
	`

	_, err := r.pool.Exec(ctx, query,
		k.ID, k.KeyPrefix, k.KeyHint, k.KeyHash, k.RootSignatureKey,
		k.Name, k.OwnerID, k.Scopes, k.ExpiresAt, k.State, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert api key: %w", err)
	}

	k.CreatedAt = now
	k.UpdatedAt = now
	return nil
}

// GetKeyByID fetches an API key by its UUID
func (r *Repository) GetKeyByID(ctx context.Context, id uuid.UUID) (*core.APIKey, error) {
	query := `
		SELECT id, key_prefix, key_hint, key_hash, root_signature_key,
		       name, owner_id, scopes, expires_at, last_used_at, state, created_at, updated_at
		FROM api_keys
		WHERE id = $1
	`

	var k core.APIKey
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&k.ID, &k.KeyPrefix, &k.KeyHint, &k.KeyHash, &k.RootSignatureKey,
		&k.Name, &k.OwnerID, &k.Scopes, &k.ExpiresAt, &k.LastUsedAt, &k.State, &k.CreatedAt, &k.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query api key by id: %w", err)
	}

	return &k, nil
}

// GetKeyByHash finds an API key by its SHA-256 hash
func (r *Repository) GetKeyByHash(ctx context.Context, keyHash string) (*core.APIKey, error) {
	query := `
		SELECT id, key_prefix, key_hint, key_hash, root_signature_key,
		       name, owner_id, scopes, expires_at, last_used_at, state, created_at, updated_at
		FROM api_keys
		WHERE key_hash = $1 AND state = 'active'
	`

	var k core.APIKey
	err := r.pool.QueryRow(ctx, query, keyHash).Scan(
		&k.ID, &k.KeyPrefix, &k.KeyHint, &k.KeyHash, &k.RootSignatureKey,
		&k.Name, &k.OwnerID, &k.Scopes, &k.ExpiresAt, &k.LastUsedAt, &k.State, &k.CreatedAt, &k.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query api key by hash: %w", err)
	}

	return &k, nil
}

// RevokeKey marks an API key as revoked
func (r *Repository) RevokeKey(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_keys SET state = 'revoked', updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke api key: %w", err)
	}
	return nil
}

// UpdateLastUsed updates the last_used_at timestamp
func (r *Repository) UpdateLastUsed(ctx context.Context, id uuid.UUID) {
	query := `UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, _ = r.pool.Exec(ctx, query, id)
}
