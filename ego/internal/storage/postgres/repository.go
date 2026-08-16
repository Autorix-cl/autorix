package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/autorix/ego/internal/core"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("record not found")
	ErrAlreadyExists = errors.New("identity with identifier already exists")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateIdentityWithPassword atomically inserts the identity, traits, credentials, and verifiable address
func (r *Repository) CreateIdentityWithPassword(ctx context.Context, traits map[string]interface{}, passwordHash string) (*core.Identity, error) {
	traitsJSON, err := json.Marshal(traits)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal traits: %w", err)
	}

	// Extract email identifier from traits
	email, ok := traits["email"].(string)
	if !ok || email == "" {
		return nil, errors.New("email trait is required")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Insert Identity
	identityID := uuid.New()
	now := time.Now()
	_, err = tx.Exec(ctx, `
		INSERT INTO identities (id, schema_id, traits, state, created_at, updated_at)
		VALUES ($1, 'default', $2, 'active', $3, $3)
	`, identityID, traitsJSON, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert identity: %w", err)
	}

	// 2. Insert Password Credential
	credData, _ := json.Marshal(map[string]string{"hashed_password": passwordHash})
	_, err = tx.Exec(ctx, `
		INSERT INTO credentials (id, identity_id, credential_type, credential_data, created_at, updated_at)
		VALUES ($1, $2, 'password', $3, $4, $4)
	`, uuid.New(), identityID, credData, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert credential: %w", err)
	}

	// 3. Insert Verifiable Address (Email)
	_, err = tx.Exec(ctx, `
		INSERT INTO identity_verifiable_addresses (id, identity_id, value, via, status, created_at)
		VALUES ($1, $2, $3, 'email', 'completed', $4)
	`, uuid.New(), identityID, email, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert verifiable address (already exists?): %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit tx: %w", err)
	}

	return &core.Identity{
		ID:        identityID,
		SchemaID:  "default",
		Traits:    traits,
		State:     "active",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// FindIdentityByIdentifier locates an identity by its email address
func (r *Repository) FindIdentityByIdentifier(ctx context.Context, identifier string) (*core.Identity, error) {
	query := `
		SELECT i.id, i.schema_id, i.traits, i.state, i.created_at, i.updated_at
		FROM identities i
		JOIN identity_verifiable_addresses iva ON i.id = iva.identity_id
		WHERE iva.value = $1
	`

	var i core.Identity
	var traitsJSON []byte
	err := r.pool.QueryRow(ctx, query, identifier).Scan(
		&i.ID, &i.SchemaID, &traitsJSON, &i.State, &i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to find identity: %w", err)
	}

	if err := json.Unmarshal(traitsJSON, &i.Traits); err != nil {
		return nil, fmt.Errorf("failed to unmarshal traits: %w", err)
	}

	return &i, nil
}

// GetPasswordCredential retrieves the hashed password for an identity
func (r *Repository) GetPasswordCredential(ctx context.Context, identityID uuid.UUID) (string, error) {
	query := `
		SELECT credential_data
		FROM credentials
		WHERE identity_id = $1 AND credential_type = 'password'
	`

	var credDataJSON []byte
	err := r.pool.QueryRow(ctx, query, identityID).Scan(&credDataJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to query credential: %w", err)
	}

	var data map[string]string
	if err := json.Unmarshal(credDataJSON, &data); err != nil {
		return "", fmt.Errorf("failed to unmarshal credential data: %w", err)
	}

	hash, ok := data["hashed_password"]
	if !ok || hash == "" {
		return "", errors.New("hashed_password field missing in credential")
	}

	return hash, nil
}

// CreateSession persists an active session
func (r *Repository) CreateSession(ctx context.Context, s *core.Session) error {
	query := `
		INSERT INTO sessions (id, identity_id, token_hash, expires_at, authenticated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, query, s.ID, s.IdentityID, s.TokenHash, s.ExpiresAt, s.AuthenticatedAt)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// GetSessionByTokenHash fetches a session and joins the associated identity
func (r *Repository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*core.Session, error) {
	query := `
		SELECT s.id, s.identity_id, s.token_hash, s.expires_at, s.authenticated_at,
		       i.schema_id, i.traits, i.state, i.created_at, i.updated_at
		FROM sessions s
		JOIN identities i ON s.identity_id = i.id
		WHERE s.token_hash = $1 AND s.expires_at > CURRENT_TIMESTAMP
	`

	var s core.Session
	var i core.Identity
	var traitsJSON []byte

	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&s.ID, &s.IdentityID, &s.TokenHash, &s.ExpiresAt, &s.AuthenticatedAt,
		&i.SchemaID, &traitsJSON, &i.State, &i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to fetch session: %w", err)
	}

	i.ID = s.IdentityID
	if err := json.Unmarshal(traitsJSON, &i.Traits); err != nil {
		return nil, fmt.Errorf("failed to unmarshal traits: %w", err)
	}
	s.Identity = &i

	return &s, nil
}

// DeleteSessionByTokenHash revokes a session
func (r *Repository) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	query := `DELETE FROM sessions WHERE token_hash = $1`
	_, err := r.pool.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}
