package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/autorix/ego/internal/core"
	"github.com/autorix/platform/paging"
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

// Pool exposes the underlying connection pool so callers (e.g. the readiness
// checker) can probe database reachability without duplicating connection
// wiring.
func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
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
	// Rollback is a no-op once Commit has succeeded (pgx returns
	// ErrTxClosed, which is expected here and intentionally ignored).
	defer func() { _ = tx.Rollback(ctx) }()

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
		SELECT i.id, i.schema_id, i.traits, i.state, i.created_at, i.updated_at, i.deleted_at
		FROM identities i
		JOIN identity_verifiable_addresses iva ON i.id = iva.identity_id
		WHERE iva.value = $1 AND i.deleted_at IS NULL
	`

	var i core.Identity
	var traitsJSON []byte
	err := r.pool.QueryRow(ctx, query, identifier).Scan(
		&i.ID, &i.SchemaID, &traitsJSON, &i.State, &i.CreatedAt, &i.UpdatedAt, &i.DeletedAt,
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

// GetIdentityByID retrieves an identity by primary key UUID (including soft-deleted ones)
func (r *Repository) GetIdentityByID(ctx context.Context, id uuid.UUID) (*core.Identity, error) {
	query := `
		SELECT id, schema_id, traits, state, created_at, updated_at, deleted_at
		FROM identities
		WHERE id = $1
	`
	var i core.Identity
	var traitsJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&i.ID, &i.SchemaID, &traitsJSON, &i.State, &i.CreatedAt, &i.UpdatedAt, &i.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get identity by id: %w", err)
	}
	if err := json.Unmarshal(traitsJSON, &i.Traits); err != nil {
		return nil, fmt.Errorf("failed to unmarshal traits: %w", err)
	}
	return &i, nil
}

// UpdateIdentity updates traits, state, or schema_id for an active identity
func (r *Repository) UpdateIdentity(ctx context.Context, id uuid.UUID, traits map[string]interface{}, state *string, schemaID *string) (*core.Identity, error) {
	if state != nil {
		switch *state {
		case core.StateActive, core.StateInactive, core.StateSuspended:
			// valid
		default:
			return nil, fmt.Errorf("invalid identity state: %s", *state)
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Fetch current identity inside tx
	var existing core.Identity
	var traitsJSON []byte
	err = tx.QueryRow(ctx, `
		SELECT id, schema_id, traits, state, created_at, updated_at, deleted_at
		FROM identities
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, id).Scan(
		&existing.ID, &existing.SchemaID, &traitsJSON, &existing.State, &existing.CreatedAt, &existing.UpdatedAt, &existing.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to load identity: %w", err)
	}
	_ = json.Unmarshal(traitsJSON, &existing.Traits)

	targetTraits := existing.Traits
	if traits != nil {
		targetTraits = traits
	}
	targetState := existing.State
	if state != nil {
		targetState = *state
	}
	targetSchemaID := existing.SchemaID
	if schemaID != nil {
		targetSchemaID = *schemaID
	}

	updatedTraitsJSON, err := json.Marshal(targetTraits)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal traits: %w", err)
	}

	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE identities
		SET schema_id = $1, traits = $2, state = $3, updated_at = $4
		WHERE id = $5
	`, targetSchemaID, updatedTraitsJSON, targetState, now, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update identity: %w", err)
	}

	// If traits has email, update or insert into identity_verifiable_addresses
	if traits != nil {
		if newEmail, ok := targetTraits["email"].(string); ok && newEmail != "" {
			_, err = tx.Exec(ctx, `
				INSERT INTO identity_verifiable_addresses (id, identity_id, value, via, status, created_at)
				VALUES ($1, $2, $3, 'email', 'completed', $4)
				ON CONFLICT (value) DO NOTHING
			`, uuid.New(), id, newEmail, now)
			if err != nil {
				return nil, fmt.Errorf("failed to upsert verifiable address: %w", err)
			}
			// Delete previous verifiable address if email changed
			oldEmail, _ := existing.Traits["email"].(string)
			if oldEmail != "" && oldEmail != newEmail {
				_, _ = tx.Exec(ctx, `
					DELETE FROM identity_verifiable_addresses
					WHERE identity_id = $1 AND value = $2
				`, id, oldEmail)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit update tx: %w", err)
	}

	return &core.Identity{
		ID:        id,
		SchemaID:  targetSchemaID,
		Traits:    targetTraits,
		State:     targetState,
		CreatedAt: existing.CreatedAt,
		UpdatedAt: now,
		DeletedAt: existing.DeletedAt,
	}, nil
}

// DeleteIdentity soft deletes an identity and revokes its sessions
func (r *Repository) DeleteIdentity(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now()
	res, err := tx.Exec(ctx, `
		UPDATE identities
		SET deleted_at = $1, state = $2, updated_at = $1
		WHERE id = $3 AND deleted_at IS NULL
	`, now, core.StateInactive, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete identity: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}

	// Revoke all active sessions for this identity
	_, err = tx.Exec(ctx, `DELETE FROM sessions WHERE identity_id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to revoke identity sessions: %w", err)
	}

	return tx.Commit(ctx)
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

	var data map[string]interface{}
	if err := json.Unmarshal(credDataJSON, &data); err != nil {
		return "", fmt.Errorf("failed to unmarshal credential data: %w", err)
	}

	hash, ok := data["hashed_password"].(string)
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
		       i.schema_id, i.traits, i.state, i.created_at, i.updated_at, i.deleted_at
		FROM sessions s
		JOIN identities i ON s.identity_id = i.id
		WHERE s.token_hash = $1 AND s.expires_at > CURRENT_TIMESTAMP AND i.deleted_at IS NULL
	`

	var s core.Session
	var i core.Identity
	var traitsJSON []byte

	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&s.ID, &s.IdentityID, &s.TokenHash, &s.ExpiresAt, &s.AuthenticatedAt,
		&i.SchemaID, &traitsJSON, &i.State, &i.CreatedAt, &i.UpdatedAt, &i.DeletedAt,
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

// ListIdentitiesFilter specifies pagination and filtering criteria for listing identities
type ListIdentitiesFilter struct {
	Limit    int
	Cursor   string
	State    string
	SchemaID string
	Traits   map[string]interface{}
	Query    string
}

// ListIdentities retrieves registered identities using real SQL keyset
// pagination with filtering by state, schema, traits (using GIN index) and query search.
func (r *Repository) ListIdentities(ctx context.Context, filter ListIdentitiesFilter) ([]core.Identity, bool, error) {
	if filter.Limit <= 0 {
		filter.Limit = paging.DefaultLimit
	}
	if filter.Limit > paging.MaxLimit {
		filter.Limit = paging.MaxLimit
	}

	query := `SELECT id, schema_id, traits, state, created_at, updated_at, deleted_at FROM identities WHERE deleted_at IS NULL`
	var args []interface{}
	argIdx := 1

	if filter.State != "" {
		query += fmt.Sprintf(" AND state = $%d", argIdx)
		args = append(args, filter.State)
		argIdx++
	}

	if filter.SchemaID != "" {
		query += fmt.Sprintf(" AND schema_id = $%d", argIdx)
		args = append(args, filter.SchemaID)
		argIdx++
	}

	if len(filter.Traits) > 0 {
		traitsJSON, err := json.Marshal(filter.Traits)
		if err != nil {
			return nil, false, fmt.Errorf("failed to marshal filter traits: %w", err)
		}
		query += fmt.Sprintf(" AND traits @> $%d::jsonb", argIdx)
		args = append(args, traitsJSON)
		argIdx++
	}

	if filter.Query != "" {
		query += fmt.Sprintf(" AND traits::text ILIKE $%d", argIdx)
		args = append(args, "%"+filter.Query+"%")
		argIdx++
	}

	if filter.Cursor != "" {
		createdAt, id, err := decodeIdentityCursor(filter.Cursor)
		if err != nil {
			return nil, false, fmt.Errorf("invalid cursor: %w", err)
		}
		query += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", argIdx, argIdx+1)
		args = append(args, createdAt, id)
		argIdx += 2
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", argIdx)
	args = append(args, filter.Limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("failed to list identities: %w", err)
	}
	defer rows.Close()

	var identities []core.Identity
	for rows.Next() {
		var i core.Identity
		var traitsJSON []byte
		if err := rows.Scan(&i.ID, &i.SchemaID, &traitsJSON, &i.State, &i.CreatedAt, &i.UpdatedAt, &i.DeletedAt); err != nil {
			return nil, false, err
		}
		_ = json.Unmarshal(traitsJSON, &i.Traits)
		identities = append(identities, i)
	}

	hasMore := len(identities) > filter.Limit
	if hasMore {
		identities = identities[:filter.Limit]
	}
	return identities, hasMore, nil
}

// decodeIdentityCursor reverses the "createdAt|id" cursor payload encoded by
// callers via paging.EncodeCursor.
func decodeIdentityCursor(cursor string) (time.Time, uuid.UUID, error) {
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

// ListActiveSessions retrieves active sessions with joined identity info using keyset pagination
func (r *Repository) ListActiveSessions(ctx context.Context, limit int, cursor string) ([]core.Session, bool, error) {
	if limit <= 0 {
		limit = paging.DefaultLimit
	}
	if limit > paging.MaxLimit {
		limit = paging.MaxLimit
	}

	query := `
		SELECT s.id, s.identity_id, s.token_hash, s.expires_at, s.authenticated_at,
		       i.schema_id, i.traits, i.state, i.created_at, i.updated_at, i.deleted_at
		FROM sessions s
		JOIN identities i ON s.identity_id = i.id
		WHERE s.expires_at > CURRENT_TIMESTAMP AND i.deleted_at IS NULL
	`
	var args []interface{}
	argIdx := 1

	if cursor != "" {
		authAt, id, err := decodeSessionCursor(cursor)
		if err != nil {
			return nil, false, fmt.Errorf("invalid cursor: %w", err)
		}
		query += fmt.Sprintf(" AND (s.authenticated_at, s.id) < ($%d, $%d)", argIdx, argIdx+1)
		args = append(args, authAt, id)
		argIdx += 2
	}

	query += fmt.Sprintf(" ORDER BY s.authenticated_at DESC, s.id DESC LIMIT $%d", argIdx)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []core.Session
	for rows.Next() {
		var s core.Session
		var i core.Identity
		var traitsJSON []byte
		if err := rows.Scan(
			&s.ID, &s.IdentityID, &s.TokenHash, &s.ExpiresAt, &s.AuthenticatedAt,
			&i.SchemaID, &traitsJSON, &i.State, &i.CreatedAt, &i.UpdatedAt, &i.DeletedAt,
		); err != nil {
			return nil, false, err
		}
		i.ID = s.IdentityID
		_ = json.Unmarshal(traitsJSON, &i.Traits)
		s.Identity = &i
		sessions = append(sessions, s)
	}

	hasMore := len(sessions) > limit
	if hasMore {
		sessions = sessions[:limit]
	}
	return sessions, hasMore, nil
}

// ListActiveSessionsByIdentity lists all non-expired sessions for a specific identity
func (r *Repository) ListActiveSessionsByIdentity(ctx context.Context, identityID uuid.UUID) ([]core.Session, error) {
	query := `
		SELECT s.id, s.identity_id, s.token_hash, s.expires_at, s.authenticated_at
		FROM sessions s
		WHERE s.identity_id = $1 AND s.expires_at > CURRENT_TIMESTAMP
		ORDER BY s.authenticated_at DESC, s.id DESC
	`
	rows, err := r.pool.Query(ctx, query, identityID)
	if err != nil {
		return nil, fmt.Errorf("failed to list identity sessions: %w", err)
	}
	defer rows.Close()

	var sessions []core.Session
	for rows.Next() {
		var s core.Session
		if err := rows.Scan(&s.ID, &s.IdentityID, &s.TokenHash, &s.ExpiresAt, &s.AuthenticatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// DeleteSessionByID deletes a single session by primary key UUID
func (r *Repository) DeleteSessionByID(ctx context.Context, sessionID uuid.UUID) error {
	query := `DELETE FROM sessions WHERE id = $1`
	res, err := r.pool.Exec(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteSessionsByIdentityID revokes all active sessions for a specific identity
func (r *Repository) DeleteSessionsByIdentityID(ctx context.Context, identityID uuid.UUID) error {
	query := `DELETE FROM sessions WHERE identity_id = $1`
	_, err := r.pool.Exec(ctx, query, identityID)
	if err != nil {
		return fmt.Errorf("failed to delete identity sessions: %w", err)
	}
	return nil
}

func decodeSessionCursor(cursor string) (time.Time, uuid.UUID, error) {
	decoded, err := paging.DecodeCursor(cursor)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	parts := strings.SplitN(decoded, "|", 2)
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, errors.New("malformed cursor payload")
	}
	authAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor timestamp: %w", err)
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor id: %w", err)
	}
	return authAt, id, nil
}

// SetPasswordCredential updates or creates the password credential for an active identity
func (r *Repository) SetPasswordCredential(ctx context.Context, identityID uuid.UUID, passwordHash string, forceRotation bool) error {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM identities WHERE id = $1 AND deleted_at IS NULL)`, identityID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check identity: %w", err)
	}
	if !exists {
		return ErrNotFound
	}

	credData, err := json.Marshal(map[string]interface{}{
		"hashed_password": passwordHash,
		"force_rotation":  forceRotation,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal credential data: %w", err)
	}

	now := time.Now()
	_, err = r.pool.Exec(ctx, `
		INSERT INTO credentials (id, identity_id, credential_type, credential_data, created_at, updated_at)
		VALUES ($1, $2, 'password', $3, $4, $4)
		ON CONFLICT (identity_id, credential_type)
		DO UPDATE SET credential_data = $3, updated_at = $4
	`, uuid.New(), identityID, credData, now)
	if err != nil {
		return fmt.Errorf("failed to set password credential: %w", err)
	}
	return nil
}

// ListCredentialsByIdentity returns safe inspection objects for all credentials of an identity
func (r *Repository) ListCredentialsByIdentity(ctx context.Context, identityID uuid.UUID) ([]core.CredentialInspection, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM identities WHERE id = $1 AND deleted_at IS NULL)`, identityID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check identity: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	query := `
		SELECT id, identity_id, credential_type, credential_data, created_at, updated_at
		FROM credentials
		WHERE identity_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, identityID)
	if err != nil {
		return nil, fmt.Errorf("failed to query credentials: %w", err)
	}
	defer rows.Close()

	var result []core.CredentialInspection
	for rows.Next() {
		var c core.CredentialInspection
		var credDataJSON []byte
		if err := rows.Scan(&c.ID, &c.IdentityID, &c.CredentialType, &credDataJSON, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		var data map[string]interface{}
		if err := json.Unmarshal(credDataJSON, &data); err == nil {
			if fr, ok := data["force_rotation"].(bool); ok {
				c.ForceRotation = fr
			}
		}
		result = append(result, c)
	}
	if result == nil {
		result = []core.CredentialInspection{}
	}
	return result, nil
}

// SaveRecoveryToken persists a recovery token for an identity
func (r *Repository) SaveRecoveryToken(ctx context.Context, identityID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM identities WHERE id = $1 AND deleted_at IS NULL)`, identityID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check identity: %w", err)
	}
	if !exists {
		return ErrNotFound
	}

	credData, err := json.Marshal(map[string]interface{}{
		"token_hash": tokenHash,
		"expires_at": expiresAt.Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal recovery token data: %w", err)
	}

	now := time.Now()
	_, err = r.pool.Exec(ctx, `
		INSERT INTO credentials (id, identity_id, credential_type, credential_data, created_at, updated_at)
		VALUES ($1, $2, 'recovery_token', $3, $4, $4)
		ON CONFLICT (identity_id, credential_type)
		DO UPDATE SET credential_data = $3, updated_at = $4
	`, uuid.New(), identityID, credData, now)
	if err != nil {
		return fmt.Errorf("failed to save recovery token: %w", err)
	}
	return nil
}

// GetRecoveryToken retrieves the stored recovery token hash and expiry
func (r *Repository) GetRecoveryToken(ctx context.Context, identityID uuid.UUID) (string, time.Time, error) {
	query := `
		SELECT credential_data
		FROM credentials
		WHERE identity_id = $1 AND credential_type = 'recovery_token'
	`
	var dataJSON []byte
	err := r.pool.QueryRow(ctx, query, identityID).Scan(&dataJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", time.Time{}, ErrNotFound
		}
		return "", time.Time{}, fmt.Errorf("failed to query recovery token: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to unmarshal recovery data: %w", err)
	}

	tokenHash, ok := data["token_hash"].(string)
	if !ok || tokenHash == "" {
		return "", time.Time{}, errors.New("token_hash missing in credential")
	}

	expStr, ok := data["expires_at"].(string)
	if !ok {
		return "", time.Time{}, errors.New("expires_at missing in credential")
	}

	exp, err := time.Parse(time.RFC3339Nano, expStr)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid expires_at format: %w", err)
	}

	return tokenHash, exp, nil
}

// SetTOTPCredential sets or replaces TOTP MFA secret and backup codes for an identity
func (r *Repository) SetTOTPCredential(ctx context.Context, identityID uuid.UUID, secret string, hashedBackupCodes []string, confirmed bool) error {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM identities WHERE id = $1 AND deleted_at IS NULL)`, identityID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check identity: %w", err)
	}
	if !exists {
		return ErrNotFound
	}

	credData, err := json.Marshal(map[string]interface{}{
		"secret":       secret,
		"backup_codes": hashedBackupCodes,
		"confirmed":    confirmed,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal totp data: %w", err)
	}

	now := time.Now()
	_, err = r.pool.Exec(ctx, `
		INSERT INTO credentials (id, identity_id, credential_type, credential_data, created_at, updated_at)
		VALUES ($1, $2, 'totp', $3, $4, $4)
		ON CONFLICT (identity_id, credential_type)
		DO UPDATE SET credential_data = $3, updated_at = $4
	`, uuid.New(), identityID, credData, now)
	if err != nil {
		return fmt.Errorf("failed to set totp credential: %w", err)
	}
	return nil
}

// GetTOTPCredential retrieves the stored TOTP secret, backup codes, and confirmation status
func (r *Repository) GetTOTPCredential(ctx context.Context, identityID uuid.UUID) (string, []string, bool, error) {
	query := `
		SELECT credential_data
		FROM credentials
		WHERE identity_id = $1 AND credential_type = 'totp'
	`
	var dataJSON []byte
	err := r.pool.QueryRow(ctx, query, identityID).Scan(&dataJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, false, ErrNotFound
		}
		return "", nil, false, fmt.Errorf("failed to query totp credential: %w", err)
	}

	var data struct {
		Secret      string   `json:"secret"`
		BackupCodes []string `json:"backup_codes"`
		Confirmed   bool     `json:"confirmed"`
	}
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return "", nil, false, fmt.Errorf("failed to unmarshal totp data: %w", err)
	}

	return data.Secret, data.BackupCodes, data.Confirmed, nil
}

// DeleteTOTPCredential removes the TOTP factor for an identity
func (r *Repository) DeleteTOTPCredential(ctx context.Context, identityID uuid.UUID) error {
	query := `DELETE FROM credentials WHERE identity_id = $1 AND credential_type = 'totp'`
	res, err := r.pool.Exec(ctx, query, identityID)
	if err != nil {
		return fmt.Errorf("failed to delete totp credential: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateSchema stores a new identity trait schema definition
func (r *Repository) CreateSchema(ctx context.Context, schema *core.IdentitySchema) (*core.IdentitySchema, error) {
	if schema.ID == "" {
		return nil, errors.New("schema ID is required")
	}
	if schema.Name == "" {
		return nil, errors.New("schema Name is required")
	}
	if schema.Schema == nil {
		return nil, errors.New("schema definition is required")
	}

	schemaJSON, err := json.Marshal(schema.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema JSON: %w", err)
	}

	now := time.Now()
	version := 1
	if schema.Version > 0 {
		version = schema.Version
	}

	query := `
		INSERT INTO identity_schemas (id, name, schema, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
	`
	_, err = r.pool.Exec(ctx, query, schema.ID, schema.Name, schemaJSON, version, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert schema: %w", err)
	}

	return &core.IdentitySchema{
		ID:        schema.ID,
		Name:      schema.Name,
		Schema:    schema.Schema,
		Version:   version,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// GetSchemaByID retrieves an identity schema by its ID string
func (r *Repository) GetSchemaByID(ctx context.Context, id string) (*core.IdentitySchema, error) {
	query := `
		SELECT id, name, schema, version, created_at, updated_at
		FROM identity_schemas
		WHERE id = $1
	`
	var s core.IdentitySchema
	var schemaJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(&s.ID, &s.Name, &schemaJSON, &s.Version, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query schema: %w", err)
	}

	if err := json.Unmarshal(schemaJSON, &s.Schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}
	return &s, nil
}

// ListSchemas returns all registered identity schemas
func (r *Repository) ListSchemas(ctx context.Context) ([]core.IdentitySchema, error) {
	query := `
		SELECT id, name, schema, version, created_at, updated_at
		FROM identity_schemas
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}
	defer rows.Close()

	var schemas []core.IdentitySchema
	for rows.Next() {
		var s core.IdentitySchema
		var schemaJSON []byte
		if err := rows.Scan(&s.ID, &s.Name, &schemaJSON, &s.Version, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(schemaJSON, &s.Schema)
		schemas = append(schemas, s)
	}
	if schemas == nil {
		schemas = []core.IdentitySchema{}
	}
	return schemas, nil
}

// UpdateSchema modifies the name or JSON definition of an identity schema, incrementing its version
func (r *Repository) UpdateSchema(ctx context.Context, id string, name *string, schema map[string]interface{}) (*core.IdentitySchema, error) {
	existing, err := r.GetSchemaByID(ctx, id)
	if err != nil {
		return nil, err
	}

	targetName := existing.Name
	if name != nil && *name != "" {
		targetName = *name
	}

	targetSchema := existing.Schema
	if schema != nil {
		targetSchema = schema
	}

	schemaJSON, err := json.Marshal(targetSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	now := time.Now()
	newVersion := existing.Version + 1

	query := `
		UPDATE identity_schemas
		SET name = $1, schema = $2, version = $3, updated_at = $4
		WHERE id = $5
	`
	_, err = r.pool.Exec(ctx, query, targetName, schemaJSON, newVersion, now, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update schema: %w", err)
	}

	return &core.IdentitySchema{
		ID:        id,
		Name:      targetName,
		Schema:    targetSchema,
		Version:   newVersion,
		CreatedAt: existing.CreatedAt,
		UpdatedAt: now,
	}, nil
}

// DeleteSchema removes an identity schema by ID
func (r *Repository) DeleteSchema(ctx context.Context, id string) error {
	query := `DELETE FROM identity_schemas WHERE id = $1`
	res, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete schema: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}



