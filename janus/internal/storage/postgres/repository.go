package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/autorix/janus/internal/core"
	"github.com/autorix/platform/paging"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

const signingKeysAdvisoryLock int64 = 0x4a4b5753 // "JKWS"

// ListSigningKeys returns the current signing key first followed by retired
// verification keys. The ordering is part of the key-manager contract.
func (r *Repository) ListSigningKeys(ctx context.Context) ([]core.SigningKey, error) {
	return listSigningKeys(ctx, r.pool)
}

// InitializeSigningKey creates the first key exactly once across all Janus
// processes. PostgreSQL's transaction-scoped advisory lock serializes the
// empty-table check and insert without requiring a second schema or a
// process-local lock.
func (r *Repository) InitializeSigningKey(ctx context.Context, candidate core.SigningKey) ([]core.SigningKey, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin signing key initialization: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", signingKeysAdvisoryLock); err != nil {
		return nil, fmt.Errorf("lock signing key initialization: %w", err)
	}
	keys, err := listSigningKeys(ctx, tx)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		if err := insertSigningKey(ctx, tx, candidate); err != nil {
			return nil, err
		}
		keys, err = listSigningKeys(ctx, tx)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit signing key initialization: %w", err)
	}
	return keys, nil
}

// RotateSigningKey persists a new RS256 signing key while retaining every
// previous key for verification rollover. Rotation is serialized so all
// instances agree on one newest current key after they reload.
func (r *Repository) RotateSigningKey(ctx context.Context, candidate core.SigningKey) ([]core.SigningKey, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin signing key rotation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", signingKeysAdvisoryLock); err != nil {
		return nil, fmt.Errorf("lock signing key rotation: %w", err)
	}
	if err := insertSigningKey(ctx, tx, candidate); err != nil {
		return nil, err
	}
	keys, err := listSigningKeys(ctx, tx)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit signing key rotation: %w", err)
	}
	return keys, nil
}

type signingKeyQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type signingKeyExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func listSigningKeys(ctx context.Context, q signingKeyQuerier) ([]core.SigningKey, error) {
	rows, err := q.Query(ctx, `
		SELECT kid, algorithm, use, private_key_pem, public_key_pem, created_at
		FROM jwks_keys
		ORDER BY created_at DESC, kid DESC`)
	if err != nil {
		return nil, fmt.Errorf("list signing keys: %w", err)
	}
	defer rows.Close()
	keys := make([]core.SigningKey, 0)
	for rows.Next() {
		var key core.SigningKey
		if err := rows.Scan(&key.Kid, &key.Algorithm, &key.Use, &key.PrivateKeyPEM, &key.PublicKeyPEM, &key.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan signing key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate signing keys: %w", err)
	}
	return keys, nil
}

func insertSigningKey(ctx context.Context, q signingKeyExecer, key core.SigningKey) error {
	if key.Kid == "" || key.Algorithm != "RS256" || key.Use != "sig" || key.PrivateKeyPEM == "" || key.PublicKeyPEM == "" {
		return errors.New("invalid signing key")
	}
	_, err := q.Exec(ctx, `
		INSERT INTO jwks_keys (kid, algorithm, use, private_key_pem, public_key_pem, created_at)
		VALUES ($1, $2, $3, $4, $5, clock_timestamp())`,
		key.Kid, key.Algorithm, key.Use, key.PrivateKeyPEM, key.PublicKeyPEM)
	if err != nil {
		return fmt.Errorf("insert signing key: %w", err)
	}
	return nil
}

// CreateClient registers a new OAuth2 client
func (r *Repository) CreateClient(ctx context.Context, client *core.OAuth2Client) error {
	now := time.Now()
	if client.GrantTypes == nil {
		client.GrantTypes = []string{}
	}
	if client.ResponseTypes == nil {
		client.ResponseTypes = []string{}
	}
	if client.RedirectURIs == nil {
		client.RedirectURIs = []string{}
	}
	if client.Scopes == nil {
		client.Scopes = []string{}
	}
	if client.AllowedAudiences == nil {
		client.AllowedAudiences = []string{}
	}

	query := `
		INSERT INTO oauth2_clients (
			id, client_name, client_secret_hash, previous_secret_hash, previous_secret_expires_at,
			grant_types, response_types, redirect_uris, scopes, allowed_audiences, is_public, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12)
	`

	_, err := r.pool.Exec(ctx, query,
		client.ID, client.ClientName, client.ClientSecretHash,
		client.PreviousSecretHash, client.PreviousSecretExpiresAt,
		client.GrantTypes, client.ResponseTypes, client.RedirectURIs,
		client.Scopes, client.AllowedAudiences, client.IsPublic, now,
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
		SELECT id, client_name, client_secret_hash, previous_secret_hash, previous_secret_expires_at,
		       grant_types, response_types, redirect_uris, scopes, allowed_audiences, is_public, created_at, updated_at
		FROM oauth2_clients
		WHERE id = $1
	`

	var c core.OAuth2Client
	var secretHash, prevSecretHash *string
	var prevExpiresAt *time.Time
	err := r.pool.QueryRow(ctx, query, clientID).Scan(
		&c.ID, &c.ClientName, &secretHash, &prevSecretHash, &prevExpiresAt,
		&c.GrantTypes, &c.ResponseTypes, &c.RedirectURIs, &c.Scopes, &c.AllowedAudiences, &c.IsPublic,
		&c.CreatedAt, &c.UpdatedAt,
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
	if prevSecretHash != nil {
		c.PreviousSecretHash = *prevSecretHash
	}
	c.PreviousSecretExpiresAt = prevExpiresAt

	return &c, nil
}

// UpdateClient updates an existing OAuth2 client's metadata
func (r *Repository) UpdateClient(ctx context.Context, client *core.OAuth2Client) error {
	now := time.Now()
	if client.GrantTypes == nil {
		client.GrantTypes = []string{}
	}
	if client.ResponseTypes == nil {
		client.ResponseTypes = []string{}
	}
	if client.RedirectURIs == nil {
		client.RedirectURIs = []string{}
	}
	if client.Scopes == nil {
		client.Scopes = []string{}
	}
	if client.AllowedAudiences == nil {
		client.AllowedAudiences = []string{}
	}

	query := `
		UPDATE oauth2_clients
		SET client_name = $2, grant_types = $3, response_types = $4,
		    redirect_uris = $5, scopes = $6, allowed_audiences = $7, is_public = $8, updated_at = $9
		WHERE id = $1
	`

	tag, err := r.pool.Exec(ctx, query,
		client.ID, client.ClientName, client.GrantTypes, client.ResponseTypes,
		client.RedirectURIs, client.Scopes, client.AllowedAudiences, client.IsPublic, now,
	)
	if err != nil {
		return fmt.Errorf("failed to update client: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	client.UpdatedAt = now
	return nil
}

// DeleteClient removes an OAuth2 client by ID
func (r *Repository) DeleteClient(ctx context.Context, clientID string) error {
	query := `DELETE FROM oauth2_clients WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, clientID)
	if err != nil {
		return fmt.Errorf("failed to delete client: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RotateClientSecret performs secret rotation with overlap window support
func (r *Repository) RotateClientSecret(ctx context.Context, clientID, newSecretHash string, overlapExpiresAt *time.Time) error {
	now := time.Now()
	query := `
		UPDATE oauth2_clients
		SET previous_secret_hash = client_secret_hash,
		    previous_secret_expires_at = $2,
		    client_secret_hash = $3,
		    updated_at = $4
		WHERE id = $1
	`

	tag, err := r.pool.Exec(ctx, query, clientID, overlapExpiresAt, newSecretHash, now)
	if err != nil {
		return fmt.Errorf("failed to rotate client secret: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateGrant stores an authorization code
func (r *Repository) CreateGrant(ctx context.Context, grant *core.Grant) error {
	now := time.Now()
	query := `
		INSERT INTO oauth2_grants (
			code_hash, client_id, subject, scopes, redirect_uri, resource,
			code_challenge, code_challenge_method, expires_at, consumed, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.pool.Exec(ctx, query,
		grant.CodeHash, grant.ClientID, grant.Subject, grant.Scopes, grant.RedirectURI, grant.Resource,
		grant.CodeChallenge, grant.CodeChallengeMethod, grant.ExpiresAt, grant.Consumed, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert grant: %w", err)
	}

	grant.CreatedAt = now
	return nil
}

// ConsumeGrant retrieves and marks an authorization code as consumed atomically
// only when it belongs to the authenticated client and redirect URI that
// initiated the authorization request. A failed binding check must not consume
// the code, allowing its legitimate client to exchange it.
func (r *Repository) ConsumeGrant(ctx context.Context, codeHash, clientID, redirectURI string) (*core.Grant, error) {
	query := `
		UPDATE oauth2_grants
		SET consumed = true
		WHERE code_hash = $1 AND client_id = $2 AND redirect_uri = $3
		  AND consumed = false AND expires_at > CURRENT_TIMESTAMP
		RETURNING client_id, subject, scopes, redirect_uri, resource, code_challenge, code_challenge_method, expires_at, consumed, created_at
	`

	var g core.Grant
	g.CodeHash = codeHash
	err := r.pool.QueryRow(ctx, query, codeHash, clientID, redirectURI).Scan(
		&g.ClientID, &g.Subject, &g.Scopes, &g.RedirectURI, &g.Resource,
		&g.CodeChallenge, &g.CodeChallengeMethod, &g.ExpiresAt, &g.Consumed, &g.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to consume grant: %w", err)
	}

	return &g, nil
}

// ListClients fetches registered OAuth2 clients using real SQL keyset pagination
func (r *Repository) ListClients(ctx context.Context, limit int, cursor string) ([]core.OAuth2Client, bool, error) {
	query := `
		SELECT id, client_name, client_secret_hash, previous_secret_hash, previous_secret_expires_at,
		       grant_types, response_types, redirect_uris, scopes, allowed_audiences, is_public, created_at, updated_at
		FROM oauth2_clients`
	args := []interface{}{}

	if cursor != "" {
		createdAt, id, err := decodeClientCursor(cursor)
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
		return nil, false, fmt.Errorf("failed to list clients: %w", err)
	}
	defer rows.Close()

	var clients []core.OAuth2Client
	for rows.Next() {
		var c core.OAuth2Client
		var secretHash, prevSecretHash *string
		var prevExpiresAt *time.Time
		if err := rows.Scan(
			&c.ID, &c.ClientName, &secretHash, &prevSecretHash, &prevExpiresAt,
			&c.GrantTypes, &c.ResponseTypes, &c.RedirectURIs, &c.Scopes, &c.AllowedAudiences, &c.IsPublic,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, false, err
		}
		if secretHash != nil {
			c.ClientSecretHash = *secretHash
		}
		if prevSecretHash != nil {
			c.PreviousSecretHash = *prevSecretHash
		}
		c.PreviousSecretExpiresAt = prevExpiresAt
		clients = append(clients, c)
	}

	hasMore := len(clients) > limit
	if hasMore {
		clients = clients[:limit]
	}
	return clients, hasMore, nil
}

// ListGrants fetches authorization grants with optional filters on client_id,
// subject, and active status, using keyset pagination.
func (r *Repository) ListGrants(ctx context.Context, clientID, subject string, activeOnly bool, limit int, cursor string) ([]core.Grant, bool, error) {
	query := `
		SELECT code_hash, client_id, subject, scopes, redirect_uri,
		       code_challenge, code_challenge_method, expires_at, consumed, created_at
		FROM oauth2_grants`
	whereClauses := []string{}
	args := []interface{}{}

	if clientID != "" {
		args = append(args, clientID)
		whereClauses = append(whereClauses, fmt.Sprintf("client_id = $%d", len(args)))
	}
	if subject != "" {
		args = append(args, subject)
		whereClauses = append(whereClauses, fmt.Sprintf("subject = $%d", len(args)))
	}
	if activeOnly {
		whereClauses = append(whereClauses, "consumed = false AND expires_at > CURRENT_TIMESTAMP")
	}
	if cursor != "" {
		createdAt, codeHash, err := decodeClientCursor(cursor)
		if err != nil {
			return nil, false, fmt.Errorf("invalid cursor: %w", err)
		}
		args = append(args, createdAt, codeHash)
		whereClauses = append(whereClauses, fmt.Sprintf("(created_at, code_hash) < ($%d, $%d)", len(args)-1, len(args)))
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += fmt.Sprintf(` ORDER BY created_at DESC, code_hash DESC LIMIT $%d`, len(args)+1)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("failed to list grants: %w", err)
	}
	defer rows.Close()

	var grants []core.Grant
	for rows.Next() {
		var g core.Grant
		var codeChallenge, codeChallengeMethod *string
		if err := rows.Scan(
			&g.CodeHash, &g.ClientID, &g.Subject, &g.Scopes, &g.RedirectURI,
			&codeChallenge, &codeChallengeMethod, &g.ExpiresAt, &g.Consumed, &g.CreatedAt,
		); err != nil {
			return nil, false, err
		}
		if codeChallenge != nil {
			g.CodeChallenge = *codeChallenge
		}
		if codeChallengeMethod != nil {
			g.CodeChallengeMethod = *codeChallengeMethod
		}
		grants = append(grants, g)
	}

	hasMore := len(grants) > limit
	if hasMore {
		grants = grants[:limit]
	}
	return grants, hasMore, nil
}

// RevokeToken marks a token as revoked, inserting a record if one doesn't exist
func (r *Repository) RevokeToken(ctx context.Context, record *core.TokenRecord) error {
	now := time.Now()
	if record.Scopes == nil {
		record.Scopes = []string{}
	}
	if record.ExpiresAt.IsZero() {
		record.ExpiresAt = now.Add(24 * time.Hour)
	}
	if record.TokenType == "" {
		record.TokenType = "access_token"
	}

	query := `
		INSERT INTO oauth2_tokens (
			token_hash, client_id, subject, token_type, scopes, expires_at, revoked, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, true, $7)
		ON CONFLICT (token_hash) DO UPDATE SET revoked = true
	`

	_, err := r.pool.Exec(ctx, query,
		record.TokenHash, record.ClientID, record.Subject,
		record.TokenType, record.Scopes, record.ExpiresAt, now,
	)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	return nil
}

// IsTokenRevoked returns true if the token is recorded and marked revoked
func (r *Repository) IsTokenRevoked(ctx context.Context, tokenHash string) (bool, error) {
	query := `SELECT revoked FROM oauth2_tokens WHERE token_hash = $1`
	var revoked bool
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(&revoked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check token revocation: %w", err)
	}
	return revoked, nil
}

// GetTokenRecord retrieves an issued token record by its hash
func (r *Repository) GetTokenRecord(ctx context.Context, tokenHash string) (*core.TokenRecord, error) {
	query := `
		SELECT token_hash, client_id, subject, token_type, scopes, expires_at, revoked, created_at
		FROM oauth2_tokens
		WHERE token_hash = $1
	`
	var rec core.TokenRecord
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&rec.TokenHash, &rec.ClientID, &rec.Subject, &rec.TokenType,
		&rec.Scopes, &rec.ExpiresAt, &rec.Revoked, &rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query token record: %w", err)
	}
	return &rec, nil
}

// CreateScope registers a new scope in the catalogue
func (r *Repository) CreateScope(ctx context.Context, scope *core.OAuth2Scope) error {
	now := time.Now()
	if scope.Claims == nil {
		scope.Claims = []string{}
	}

	query := `
		INSERT INTO oauth2_scopes (name, description, claims, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.pool.Exec(ctx, query, scope.Name, scope.Description, scope.Claims, now)
	if err != nil {
		return fmt.Errorf("failed to insert scope: %w", err)
	}
	scope.CreatedAt = now
	return nil
}

// GetScope retrieves a scope definition by name
func (r *Repository) GetScope(ctx context.Context, name string) (*core.OAuth2Scope, error) {
	query := `SELECT name, description, claims, created_at FROM oauth2_scopes WHERE name = $1`
	var s core.OAuth2Scope
	err := r.pool.QueryRow(ctx, query, name).Scan(&s.Name, &s.Description, &s.Claims, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query scope: %w", err)
	}
	return &s, nil
}

// ListScopes lists all scopes registered in the catalogue
func (r *Repository) ListScopes(ctx context.Context) ([]core.OAuth2Scope, error) {
	query := `SELECT name, description, claims, created_at FROM oauth2_scopes ORDER BY created_at ASC, name ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list scopes: %w", err)
	}
	defer rows.Close()

	var scopes []core.OAuth2Scope
	for rows.Next() {
		var s core.OAuth2Scope
		if err := rows.Scan(&s.Name, &s.Description, &s.Claims, &s.CreatedAt); err != nil {
			return nil, err
		}
		scopes = append(scopes, s)
	}
	return scopes, nil
}

// DeleteScope removes a scope from the catalogue
func (r *Repository) DeleteScope(ctx context.Context, name string) error {
	query := `DELETE FROM oauth2_scopes WHERE name = $1`
	tag, err := r.pool.Exec(ctx, query, name)
	if err != nil {
		return fmt.Errorf("failed to delete scope: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// decodeClientCursor reverses the "createdAt|id" cursor payload encoded by
// callers via paging.EncodeCursor.
func decodeClientCursor(cursor string) (time.Time, string, error) {
	decoded, err := paging.DecodeCursor(cursor)
	if err != nil {
		return time.Time{}, "", err
	}
	parts := strings.SplitN(decoded, "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", errors.New("malformed cursor payload")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor timestamp: %w", err)
	}
	return createdAt, parts[1], nil
}

// CreateLoginChallenge stores a new login challenge
func (r *Repository) CreateLoginChallenge(ctx context.Context, challenge *core.LoginChallenge) error {
	now := time.Now()
	query := `
		INSERT INTO oauth2_login_challenges (
			challenge, client_id, redirect_uri, response_type, scopes, resource, state, nonce,
			code_challenge, code_challenge_method, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, query,
		challenge.Challenge, challenge.ClientID, challenge.RedirectURI, challenge.ResponseType,
		challenge.Scopes, challenge.Resource, challenge.State, challenge.Nonce, challenge.CodeChallenge,
		challenge.CodeChallengeMethod, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert login challenge: %w", err)
	}
	challenge.CreatedAt = now
	return nil
}

// GetLoginChallenge retrieves a login challenge
func (r *Repository) GetLoginChallenge(ctx context.Context, challenge string) (*core.LoginChallenge, error) {
	query := `
		SELECT challenge, client_id, redirect_uri, response_type, scopes, resource, state, nonce,
		       code_challenge, code_challenge_method, subject, login_verifier, handled_at, created_at
		FROM oauth2_login_challenges
		WHERE challenge = $1
	`
	var c core.LoginChallenge
	var subject, loginVerifier *string
	err := r.pool.QueryRow(ctx, query, challenge).Scan(
		&c.Challenge, &c.ClientID, &c.RedirectURI, &c.ResponseType, &c.Scopes, &c.Resource, &c.State, &c.Nonce,
		&c.CodeChallenge, &c.CodeChallengeMethod, &subject, &loginVerifier, &c.HandledAt, &c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query login challenge: %w", err)
	}
	if subject != nil {
		c.Subject = *subject
	}
	if loginVerifier != nil {
		c.LoginVerifier = *loginVerifier
	}
	return &c, nil
}

// UpdateLoginChallenge updates a login challenge (e.g., when accepted)
func (r *Repository) UpdateLoginChallenge(ctx context.Context, c *core.LoginChallenge) error {
	query := `
		UPDATE oauth2_login_challenges
		SET subject = $2, login_verifier = $3, handled_at = $4
		WHERE challenge = $1
	`
	tag, err := r.pool.Exec(ctx, query, c.Challenge, c.Subject, c.LoginVerifier, c.HandledAt)
	if err != nil {
		return fmt.Errorf("failed to update login challenge: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateConsentChallenge stores a new consent challenge
func (r *Repository) CreateConsentChallenge(ctx context.Context, challenge *core.ConsentChallenge) error {
	now := time.Now()
	query := `
		INSERT INTO oauth2_consent_challenges (
			challenge, login_challenge, client_id, subject, requested_scopes, resource, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		challenge.Challenge, challenge.LoginChallenge, challenge.ClientID, challenge.Subject,
		challenge.RequestedScopes, challenge.Resource, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert consent challenge: %w", err)
	}
	challenge.CreatedAt = now
	return nil
}

// GetConsentChallenge retrieves a consent challenge
func (r *Repository) GetConsentChallenge(ctx context.Context, challenge string) (*core.ConsentChallenge, error) {
	query := `
		SELECT challenge, login_challenge, client_id, subject, requested_scopes, resource, granted_scopes,
		       consent_verifier, handled_at, created_at
		FROM oauth2_consent_challenges
		WHERE challenge = $1
	`
	var c core.ConsentChallenge
	var consentVerifier *string
	err := r.pool.QueryRow(ctx, query, challenge).Scan(
		&c.Challenge, &c.LoginChallenge, &c.ClientID, &c.Subject, &c.RequestedScopes, &c.Resource, &c.GrantedScopes,
		&consentVerifier, &c.HandledAt, &c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query consent challenge: %w", err)
	}
	if consentVerifier != nil {
		c.ConsentVerifier = *consentVerifier
	}
	return &c, nil
}

// UpdateConsentChallenge updates a consent challenge (e.g., when accepted)
func (r *Repository) UpdateConsentChallenge(ctx context.Context, c *core.ConsentChallenge) error {
	query := `
		UPDATE oauth2_consent_challenges
		SET granted_scopes = $2, consent_verifier = $3, handled_at = $4
		WHERE challenge = $1
	`
	tag, err := r.pool.Exec(ctx, query, c.Challenge, c.GrantedScopes, c.ConsentVerifier, c.HandledAt)
	if err != nil {
		return fmt.Errorf("failed to update consent challenge: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
