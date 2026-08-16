package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/autorix/janus/internal/core"
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

// CreateClient registers a new OAuth2 client
func (r *Repository) CreateClient(ctx context.Context, client *core.OAuth2Client) error {
	now := time.Now()
	query := `
		INSERT INTO oauth2_clients (
			id, client_name, client_secret_hash, grant_types, response_types,
			redirect_uris, scopes, is_public, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
	`

	_, err := r.pool.Exec(ctx, query,
		client.ID, client.ClientName, client.ClientSecretHash,
		client.GrantTypes, client.ResponseTypes, client.RedirectURIs,
		client.Scopes, client.IsPublic, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert oauth2 client: %w", err)
	}

	client.CreatedAt = now
	client.UpdatedAt = now
	return nil
}

// GetClient retrieves an OAuth2 client by ID
func (r *Repository) GetClient(ctx context.Context, clientID string) (*core.OAuth2Client, error) {
	query := `
		SELECT id, client_name, client_secret_hash, grant_types, response_types,
		       redirect_uris, scopes, is_public, created_at, updated_at
		FROM oauth2_clients
		WHERE id = $1
	`

	var c core.OAuth2Client
	var secretHash *string
	err := r.pool.QueryRow(ctx, query, clientID).Scan(
		&c.ID, &c.ClientName, &secretHash, &c.GrantTypes, &c.ResponseTypes,
		&c.RedirectURIs, &c.Scopes, &c.IsPublic, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query client: %w", err)
	}

	if secretHash != nil {
		c.ClientSecretHash = *secretHash
	}

	return &c, nil
}

// CreateGrant stores an authorization code
func (r *Repository) CreateGrant(ctx context.Context, grant *core.Grant) error {
	query := `
		INSERT INTO oauth2_grants (
			code_hash, client_id, subject, scopes, redirect_uri,
			code_challenge, code_challenge_method, expires_at, consumed
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.pool.Exec(ctx, query,
		grant.CodeHash, grant.ClientID, grant.Subject, grant.Scopes, grant.RedirectURI,
		grant.CodeChallenge, grant.CodeChallengeMethod, grant.ExpiresAt, grant.Consumed,
	)
	if err != nil {
		return fmt.Errorf("failed to insert grant: %w", err)
	}

	return nil
}

// ConsumeGrant retrieves and marks an authorization code as consumed atomically
func (r *Repository) ConsumeGrant(ctx context.Context, codeHash string) (*core.Grant, error) {
	query := `
		UPDATE oauth2_grants
		SET consumed = true
		WHERE code_hash = $1 AND consumed = false AND expires_at > CURRENT_TIMESTAMP
		RETURNING client_id, subject, scopes, redirect_uri, code_challenge, code_challenge_method, expires_at, consumed
	`

	var g core.Grant
	g.CodeHash = codeHash
	err := r.pool.QueryRow(ctx, query, codeHash).Scan(
		&g.ClientID, &g.Subject, &g.Scopes, &g.RedirectURI,
		&g.CodeChallenge, &g.CodeChallengeMethod, &g.ExpiresAt, &g.Consumed,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to consume grant: %w", err)
	}

	return &g, nil
}
