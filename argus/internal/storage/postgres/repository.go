// Package postgres is Argus's Postgres adapter (P2-S2): the concrete
// implementation of core.Repository against the tables created by
// migrations/000001_create_registry_tables.up.sql. It owns all SQL for the
// registry — nothing outside this package talks to the database directly.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/autorix/argus/internal/core"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// querier is satisfied by both *pgxpool.Pool and pgx.Tx, so the row-mapping
// helpers below work identically whether they run standalone or inside a
// transaction (used by the transactional writers: SetInstanceStatusWithEvent
// and RegisterInstance).
type querier interface {
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

type Repository struct {
	pool *pgxpool.Pool
}

// Compile-time guard: Repository must keep implementing core.Repository.
var _ core.Repository = (*Repository)(nil)

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Pool exposes the underlying connection pool so main.go can wire
// postgres.Check into the engine's health.Checker (ADR 0001).
func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}

// ---------------------------------------------------------------------
// Environments
// ---------------------------------------------------------------------

func (r *Repository) CreateEnvironment(ctx context.Context, name, slug, description string, isProduction bool) (core.Environment, error) {
	var env core.Environment
	var projectID *uuid.UUID
	var envType string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO environments (name, slug, description, is_production)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, slug, description, is_production, project_id, type, created_at
	`, name, slug, description, isProduction).Scan(
		&env.ID, &env.Name, &env.Slug, &env.Description, &env.IsProduction, &projectID, &envType, &env.CreatedAt,
	)
	if err != nil {
		return core.Environment{}, fmt.Errorf("creating environment: %w", err)
	}
	env.ProjectID = projectID
	env.Type = envType
	return env, nil
}

func (r *Repository) GetEnvironmentBySlug(ctx context.Context, slug string) (core.Environment, error) {
	var env core.Environment
	var projectID *uuid.UUID
	var envType string
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, description, is_production, project_id, type, created_at
		FROM environments WHERE slug = $1
	`, slug).Scan(&env.ID, &env.Name, &env.Slug, &env.Description, &env.IsProduction, &projectID, &envType, &env.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return core.Environment{}, core.ErrNotFound
	}
	if err != nil {
		return core.Environment{}, fmt.Errorf("getting environment by slug: %w", err)
	}
	env.ProjectID = projectID
	env.Type = envType
	return env, nil
}

func (r *Repository) GetEnvironmentByID(ctx context.Context, id uuid.UUID) (core.Environment, error) {
	var env core.Environment
	var projectID *uuid.UUID
	var envType string
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, description, is_production, project_id, type, created_at
		FROM environments WHERE id = $1
	`, id).Scan(&env.ID, &env.Name, &env.Slug, &env.Description, &env.IsProduction, &projectID, &envType, &env.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return core.Environment{}, core.ErrNotFound
	}
	if err != nil {
		return core.Environment{}, fmt.Errorf("getting environment by id: %w", err)
	}
	env.ProjectID = projectID
	env.Type = envType
	return env, nil
}

func (r *Repository) ListEnvironments(ctx context.Context) ([]core.Environment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, slug, description, is_production, project_id, type, created_at
		FROM environments ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("listing environments: %w", err)
	}
	defer rows.Close()

	var out []core.Environment
	for rows.Next() {
		var env core.Environment
		var projectID *uuid.UUID
		var envType string
		if err := rows.Scan(&env.ID, &env.Name, &env.Slug, &env.Description, &env.IsProduction, &projectID, &envType, &env.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning environment: %w", err)
		}
		env.ProjectID = projectID
		env.Type = envType
		out = append(out, env)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------
// Engine types (seeded, read-mostly)
// ---------------------------------------------------------------------

func (r *Repository) ListEngineTypes(ctx context.Context) ([]core.EngineType, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT slug, display_name, default_rest_port, default_grpc_port, protocol, capability_schema
		FROM engine_types ORDER BY slug
	`)
	if err != nil {
		return nil, fmt.Errorf("listing engine types: %w", err)
	}
	defer rows.Close()

	var out []core.EngineType
	for rows.Next() {
		et, err := scanEngineType(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, et)
	}
	return out, rows.Err()
}

func (r *Repository) GetEngineType(ctx context.Context, slug string) (core.EngineType, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT slug, display_name, default_rest_port, default_grpc_port, protocol, capability_schema
		FROM engine_types WHERE slug = $1
	`, slug)
	et, err := scanEngineType(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return core.EngineType{}, core.ErrNotFound
	}
	if err != nil {
		return core.EngineType{}, fmt.Errorf("getting engine type: %w", err)
	}
	return et, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanEngineType(row rowScanner) (core.EngineType, error) {
	var et core.EngineType
	var capSchema []byte
	if err := row.Scan(&et.Slug, &et.DisplayName, &et.DefaultRESTPort, &et.DefaultGRPCPort, &et.Protocol, &capSchema); err != nil {
		return core.EngineType{}, err
	}
	if len(capSchema) > 0 {
		if err := json.Unmarshal(capSchema, &et.CapabilitySchema); err != nil {
			return core.EngineType{}, fmt.Errorf("unmarshaling capability_schema: %w", err)
		}
	}
	return et, nil
}

// ---------------------------------------------------------------------
// Enrollment tokens (P2-S3)
// ---------------------------------------------------------------------

func (r *Repository) CreateEnrollmentToken(ctx context.Context, t core.EnrollmentToken) (core.EnrollmentToken, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO enrollment_tokens (token_hash, engine_type, environment_id, uses_allowed, expires_at, created_by)
		VALUES ($1, $2, $3, NULLIF($4, 0), $5, $6)
		RETURNING id, token_hash, engine_type, environment_id, uses_allowed, uses_consumed, expires_at, created_by, created_at, revoked_at
	`, t.TokenHash, t.EngineType, t.EnvironmentID, t.UsesAllowed, t.ExpiresAt, t.CreatedBy).Scan(
		&t.ID, &t.TokenHash, &t.EngineType, &t.EnvironmentID, &t.UsesAllowed, &t.UsesConsumed,
		&t.ExpiresAt, &t.CreatedBy, &t.CreatedAt, &t.RevokedAt,
	)
	if err != nil {
		return core.EnrollmentToken{}, fmt.Errorf("creating enrollment token: %w", err)
	}
	return t, nil
}

func (r *Repository) GetEnrollmentTokenByHash(ctx context.Context, tokenHash string) (core.EnrollmentToken, error) {
	var t core.EnrollmentToken
	err := r.pool.QueryRow(ctx, `
		SELECT id, token_hash, engine_type, environment_id, uses_allowed, uses_consumed, expires_at, created_by, created_at, revoked_at
		FROM enrollment_tokens WHERE token_hash = $1
	`, tokenHash).Scan(
		&t.ID, &t.TokenHash, &t.EngineType, &t.EnvironmentID, &t.UsesAllowed, &t.UsesConsumed,
		&t.ExpiresAt, &t.CreatedBy, &t.CreatedAt, &t.RevokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return core.EnrollmentToken{}, core.ErrEnrollmentTokenInvalid
	}
	if err != nil {
		return core.EnrollmentToken{}, fmt.Errorf("getting enrollment token by hash: %w", err)
	}
	return t, nil
}

func (r *Repository) ListEnrollmentTokens(ctx context.Context, engineType string, environmentID uuid.UUID) ([]core.EnrollmentToken, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, token_hash, engine_type, environment_id, uses_allowed, uses_consumed, expires_at, created_by, created_at, revoked_at
		FROM enrollment_tokens
		WHERE ($1 = '' OR engine_type = $1) AND ($2 = '00000000-0000-0000-0000-000000000000'::uuid OR environment_id = $2)
		ORDER BY created_at DESC
	`, engineType, environmentID)
	if err != nil {
		return nil, fmt.Errorf("listing enrollment tokens: %w", err)
	}
	defer rows.Close()

	var out []core.EnrollmentToken
	for rows.Next() {
		var t core.EnrollmentToken
		if err := rows.Scan(&t.ID, &t.TokenHash, &t.EngineType, &t.EnvironmentID, &t.UsesAllowed, &t.UsesConsumed,
			&t.ExpiresAt, &t.CreatedBy, &t.CreatedAt, &t.RevokedAt); err != nil {
			return nil, fmt.Errorf("scanning enrollment token: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ConsumeEnrollmentToken atomically validates and consumes one use of a
// token: it must exist, be unrevoked, unexpired, and under its
// uses_allowed ceiling. The check and the increment happen in the same
// UPDATE so two concurrent Register calls racing on the last remaining use
// cannot both succeed.
func (r *Repository) ConsumeEnrollmentToken(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE enrollment_tokens
		SET uses_consumed = uses_consumed + 1
		WHERE id = $1
		  AND revoked_at IS NULL
		  AND expires_at > now()
		  AND uses_consumed < uses_allowed
	`, id)
	if err != nil {
		return fmt.Errorf("consuming enrollment token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core.ErrEnrollmentTokenInvalid
	}
	return nil
}

func (r *Repository) RevokeEnrollmentToken(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE enrollment_tokens SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("revoking enrollment token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core.ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------------
// Instances
// ---------------------------------------------------------------------

const instanceColumns = `id, engine_type, instance_id, environment_id, version, build_sha, schema_version,
	endpoints, capabilities, labels, status, first_seen_at, last_heartbeat_at, deregistered_at, enrolled_by, unverified`

func scanInstance(row rowScanner) (core.Instance, error) {
	var inst core.Instance
	var endpoints, capabilities, labels []byte
	if err := row.Scan(
		&inst.ID, &inst.EngineType, &inst.InstanceID, &inst.EnvironmentID, &inst.Version, &inst.BuildSHA, &inst.SchemaVersion,
		&endpoints, &capabilities, &labels, &inst.Status, &inst.FirstSeenAt, &inst.LastHeartbeatAt, &inst.DeregisteredAt,
		&inst.EnrolledBy, &inst.Unverified,
	); err != nil {
		return core.Instance{}, err
	}
	if len(endpoints) > 0 {
		if err := json.Unmarshal(endpoints, &inst.Endpoints); err != nil {
			return core.Instance{}, fmt.Errorf("unmarshaling endpoints: %w", err)
		}
	}
	if len(capabilities) > 0 {
		if err := json.Unmarshal(capabilities, &inst.Capabilities); err != nil {
			return core.Instance{}, fmt.Errorf("unmarshaling capabilities: %w", err)
		}
	}
	if len(labels) > 0 {
		if err := json.Unmarshal(labels, &inst.Labels); err != nil {
			return core.Instance{}, fmt.Errorf("unmarshaling labels: %w", err)
		}
	}
	return inst, nil
}

// upsertInstance runs the identity upsert against q (a pool or a tx), so it
// backs both the plain UpsertInstance interface method and the
// transactional RegisterInstance writer. It always inserts new rows with
// the given initialStatus, and on conflict leaves the existing status
// column untouched — a re-registering instance keeps whatever lifecycle
// state the evaluator/heartbeat path already put it in.
func upsertInstance(ctx context.Context, q querier, req core.RegistrationRequest, enrolledBy string, unverified bool, initialStatus core.InstanceStatus) (core.Instance, error) {
	endpoints, err := json.Marshal(req.Endpoints)
	if err != nil {
		return core.Instance{}, fmt.Errorf("marshaling endpoints: %w", err)
	}
	capabilities := req.Capabilities
	if capabilities == nil {
		capabilities = []string{}
	}
	capsJSON, err := json.Marshal(capabilities)
	if err != nil {
		return core.Instance{}, fmt.Errorf("marshaling capabilities: %w", err)
	}

	row := q.QueryRow(ctx, `
		INSERT INTO instances (
			engine_type, instance_id, environment_id, version, build_sha, schema_version,
			endpoints, capabilities, status, enrolled_by, unverified
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (engine_type, instance_id, environment_id) DO UPDATE SET
			version = EXCLUDED.version,
			build_sha = EXCLUDED.build_sha,
			schema_version = EXCLUDED.schema_version,
			endpoints = EXCLUDED.endpoints,
			capabilities = EXCLUDED.capabilities,
			enrolled_by = EXCLUDED.enrolled_by,
			unverified = EXCLUDED.unverified,
			updated_at = now()
		RETURNING `+instanceColumns,
		req.EngineType, req.InstanceID, req.EnvironmentID, req.Version, req.BuildSHA, req.SchemaVersion,
		endpoints, capsJSON, initialStatus, enrolledBy, unverified,
	)
	return scanInstance(row)
}

func (r *Repository) UpsertInstance(ctx context.Context, req core.RegistrationRequest, enrolledBy string, unverified bool) (core.Instance, error) {
	inst, err := upsertInstance(ctx, r.pool, req, enrolledBy, unverified, core.StatusPending)
	if err != nil {
		return core.Instance{}, fmt.Errorf("upserting instance: %w", err)
	}
	return inst, nil
}

func (r *Repository) GetInstance(ctx context.Context, id uuid.UUID) (core.Instance, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+instanceColumns+` FROM instances WHERE id = $1`, id)
	inst, err := scanInstance(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return core.Instance{}, core.ErrNotFound
	}
	if err != nil {
		return core.Instance{}, fmt.Errorf("getting instance: %w", err)
	}
	return inst, nil
}

func (r *Repository) ListInstances(ctx context.Context, filter core.InstanceFilter) ([]core.Instance, string, bool, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT ` + instanceColumns + ` FROM instances WHERE 1=1`
	var args []interface{}
	arg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if filter.EnvironmentID != uuid.Nil {
		query += " AND environment_id = " + arg(filter.EnvironmentID)
	}
	if filter.EngineType != "" {
		query += " AND engine_type = " + arg(filter.EngineType)
	}
	if filter.Status != "" {
		query += " AND status = " + arg(filter.Status)
	}
	if filter.Cursor != "" {
		cursorID, err := uuid.Parse(filter.Cursor)
		if err != nil {
			return nil, "", false, fmt.Errorf("parsing cursor: %w", err)
		}
		query += " AND id > " + arg(cursorID)
	}
	query += " ORDER BY id LIMIT " + arg(limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", false, fmt.Errorf("listing instances: %w", err)
	}
	defer rows.Close()

	var out []core.Instance
	for rows.Next() {
		inst, err := scanInstance(rows)
		if err != nil {
			return nil, "", false, fmt.Errorf("scanning instance: %w", err)
		}
		out = append(out, inst)
	}
	if err := rows.Err(); err != nil {
		return nil, "", false, err
	}

	hasMore := len(out) > limit
	if hasMore {
		out = out[:limit]
	}
	nextCursor := ""
	if hasMore && len(out) > 0 {
		nextCursor = out[len(out)-1].ID.String()
	}
	return out, nextCursor, hasMore, nil
}

func (r *Repository) SetInstanceStatus(ctx context.Context, id uuid.UUID, status core.InstanceStatus) error {
	tag, err := r.pool.Exec(ctx, `UPDATE instances SET status = $2, updated_at = now() WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("setting instance status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core.ErrNotFound
	}
	return nil
}

func (r *Repository) RecordHeartbeat(ctx context.Context, id uuid.UUID, report core.HeartbeatReport) error {
	receivedAt := report.ReceivedAt
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	tag, err := r.pool.Exec(ctx, `UPDATE instances SET last_heartbeat_at = $2, updated_at = now() WHERE id = $1`, id, receivedAt)
	if err != nil {
		return fmt.Errorf("recording heartbeat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core.ErrNotFound
	}
	return nil
}

func (r *Repository) Deregister(ctx context.Context, id uuid.UUID, reason string) error {
	return r.SetInstanceStatusWithEvent(ctx, id, core.StatusDeregistered, core.EventDeregistered, map[string]interface{}{"reason": reason})
}

func (r *Repository) ForceRemove(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM instances WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("force-removing instance: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core.ErrNotFound
	}
	return nil
}

func (r *Repository) ListStaleInstances(ctx context.Context, olderThan time.Time) ([]core.Instance, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+instanceColumns+` FROM instances
		WHERE last_heartbeat_at IS NOT NULL
		  AND last_heartbeat_at < $1
		  AND status NOT IN ($2, $3)
	`, olderThan, core.StatusEvicted, core.StatusDeregistered)
	if err != nil {
		return nil, fmt.Errorf("listing stale instances: %w", err)
	}
	defer rows.Close()

	var out []core.Instance
	for rows.Next() {
		inst, err := scanInstance(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning instance: %w", err)
		}
		out = append(out, inst)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------
// Instance credentials (P2-S3-T3)
// ---------------------------------------------------------------------

func setInstanceCredential(ctx context.Context, q querier, instanceID uuid.UUID, cred core.InstanceCredentialMaterial) error {
	_, err := q.Exec(ctx, `
		INSERT INTO instance_credentials (instance_id, secret_hash, secret_encrypted)
		VALUES ($1, $2, $3)
		ON CONFLICT (instance_id) DO UPDATE SET
			secret_hash = EXCLUDED.secret_hash,
			secret_encrypted = EXCLUDED.secret_encrypted,
			rotated_at = now(),
			revoked_at = NULL
	`, instanceID, cred.SecretHash, cred.SecretEncrypted)
	if err != nil {
		return fmt.Errorf("setting instance credential: %w", err)
	}
	return nil
}

func (r *Repository) SetInstanceCredential(ctx context.Context, instanceID uuid.UUID, cred core.InstanceCredentialMaterial) error {
	return setInstanceCredential(ctx, r.pool, instanceID, cred)
}

func (r *Repository) RotateInstanceCredential(ctx context.Context, instanceID uuid.UUID, newCred core.InstanceCredentialMaterial, overlapUntil time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE instance_credentials
		SET previous_secret_hash = secret_hash,
			previous_secret_encrypted = secret_encrypted,
			previous_secret_valid_until = $4,
			secret_hash = $2,
			secret_encrypted = $3,
			rotated_at = now()
		WHERE instance_id = $1 AND revoked_at IS NULL
	`, instanceID, newCred.SecretHash, newCred.SecretEncrypted, overlapUntil)
	if err != nil {
		return fmt.Errorf("rotating instance credential: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core.ErrNotFound
	}
	return nil
}

func (r *Repository) GetInstanceCredential(ctx context.Context, instanceID uuid.UUID) (core.InstanceCredentialRecord, error) {
	var rec core.InstanceCredentialRecord
	var prevHash *string
	dbErr := r.pool.QueryRow(ctx, `
		SELECT secret_hash, secret_encrypted, previous_secret_hash, previous_secret_encrypted, previous_secret_valid_until
		FROM instance_credentials
		WHERE instance_id = $1 AND revoked_at IS NULL
	`, instanceID).Scan(&rec.SecretHash, &rec.SecretEncrypted, &prevHash, &rec.PreviousSecretEncrypted, &rec.PreviousValidUntil)
	if errors.Is(dbErr, pgx.ErrNoRows) {
		return core.InstanceCredentialRecord{}, core.ErrNotFound
	}
	if dbErr != nil {
		return core.InstanceCredentialRecord{}, fmt.Errorf("getting instance credential: %w", dbErr)
	}
	if prevHash != nil {
		rec.PreviousSecretHash = *prevHash
	}
	return rec, nil
}

func (r *Repository) RevokeInstanceCredential(ctx context.Context, instanceID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE instance_credentials SET revoked_at = now() WHERE instance_id = $1 AND revoked_at IS NULL`, instanceID)
	if err != nil {
		return fmt.Errorf("revoking instance credential: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core.ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------------
// Events (P2-S2-T4: append-only, same transaction as the state change)
// ---------------------------------------------------------------------

func appendEvent(ctx context.Context, q querier, instanceID uuid.UUID, eventType core.InstanceEventType, detail map[string]interface{}) error {
	detailJSON, err := json.Marshal(detail)
	if err != nil {
		return fmt.Errorf("marshaling event detail: %w", err)
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO instance_events (instance_id, event_type, detail)
		VALUES ($1, $2, $3)
	`, instanceID, eventType, detailJSON); err != nil {
		return fmt.Errorf("appending event: %w", err)
	}
	return nil
}

func (r *Repository) AppendEvent(ctx context.Context, instanceID uuid.UUID, eventType core.InstanceEventType, detail map[string]interface{}) error {
	return appendEvent(ctx, r.pool, instanceID, eventType, detail)
}

func (r *Repository) ListEvents(ctx context.Context, instanceID *uuid.UUID, limit int) ([]core.InstanceEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, instance_id, event_type, detail, occurred_at FROM instance_events`
	var args []interface{}
	if instanceID != nil {
		query += ` WHERE instance_id = $1`
		args = append(args, *instanceID)
	}
	query += fmt.Sprintf(` ORDER BY occurred_at DESC LIMIT $%d`, len(args)+1)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing events: %w", err)
	}
	defer rows.Close()

	var out []core.InstanceEvent
	for rows.Next() {
		var ev core.InstanceEvent
		var detail []byte
		if err := rows.Scan(&ev.ID, &ev.InstanceID, &ev.Type, &detail, &ev.OccurredAt); err != nil {
			return nil, fmt.Errorf("scanning event: %w", err)
		}
		if len(detail) > 0 {
			if err := json.Unmarshal(detail, &ev.Detail); err != nil {
				return nil, fmt.Errorf("unmarshaling event detail: %w", err)
			}
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

// SetInstanceStatusWithEvent is the transactional lifecycle writer
// (P2-S2-T4): the status column and its append-only timeline entry are
// written in the same transaction, so the two can never disagree — a crash
// between them rolls back both instead of leaving an event with no matching
// status, or a status with no explaining event.
func (r *Repository) SetInstanceStatusWithEvent(ctx context.Context, id uuid.UUID, status core.InstanceStatus, eventType core.InstanceEventType, detail map[string]interface{}) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op once Commit has succeeded

	tag, err := tx.Exec(ctx, `UPDATE instances SET status = $2, updated_at = now() WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("updating instance status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core.ErrNotFound
	}

	if err := appendEvent(ctx, tx, id, eventType, detail); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// RegisterInstance is the transactional writer backing the gRPC Register
// RPC (P2-S3): the identity upsert, the "registering" status, the freshly
// minted credential hash, and the "registered" timeline event all commit or
// roll back together, so a crash mid-registration can never leave an
// instance with a credential but no event, or vice versa. The enrollment
// token itself is validated and consumed by the caller before this runs —
// token consumption is a separate row from instance identity, so it does
// not need to share this transaction.
func (r *Repository) RegisterInstance(ctx context.Context, req core.RegistrationRequest, enrolledBy string, unverified bool, cred core.InstanceCredentialMaterial) (core.Instance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return core.Instance{}, fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op once Commit has succeeded

	inst, err := upsertInstance(ctx, tx, req, enrolledBy, unverified, core.StatusRegistering)
	if err != nil {
		return core.Instance{}, fmt.Errorf("upserting instance: %w", err)
	}
	// A re-registration must also move back to "registering" — upsertInstance
	// only sets the initial status on INSERT, so force it explicitly here.
	if _, err := tx.Exec(ctx, `UPDATE instances SET status = $2, updated_at = now() WHERE id = $1`, inst.ID, core.StatusRegistering); err != nil {
		return core.Instance{}, fmt.Errorf("setting registering status: %w", err)
	}
	inst.Status = core.StatusRegistering

	if err := setInstanceCredential(ctx, tx, inst.ID, cred); err != nil {
		return core.Instance{}, err
	}

	if err := appendEvent(ctx, tx, inst.ID, core.EventRegistered, map[string]interface{}{
		"engine_type": req.EngineType,
		"instance_id": req.InstanceID,
		"version":     req.Version,
	}); err != nil {
		return core.Instance{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return core.Instance{}, fmt.Errorf("committing registration: %w", err)
	}
	return inst, nil
}

// ---------------------------------------------------------------------
// Dependencies (P2-S6-T3 topology)
// ---------------------------------------------------------------------

func (r *Repository) DeclareDependency(ctx context.Context, instanceID uuid.UUID, dependsOnEngineType string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO instance_dependencies (instance_id, depends_on_engine_type, declared)
		VALUES ($1, $2, TRUE)
		ON CONFLICT (instance_id, depends_on_engine_type) DO UPDATE SET declared = TRUE
	`, instanceID, dependsOnEngineType)
	if err != nil {
		return fmt.Errorf("declaring dependency: %w", err)
	}
	return nil
}

func (r *Repository) RecordDependencyProbe(ctx context.Context, instanceID uuid.UUID, dependsOnEngineType string, reachable bool, latencyMs float64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO instance_dependencies (instance_id, depends_on_engine_type, declared, last_probe_reachable, last_probe_latency_ms, last_probed_at)
		VALUES ($1, $2, FALSE, $3, $4, now())
		ON CONFLICT (instance_id, depends_on_engine_type) DO UPDATE SET
			last_probe_reachable = EXCLUDED.last_probe_reachable,
			last_probe_latency_ms = EXCLUDED.last_probe_latency_ms,
			last_probed_at = now()
	`, instanceID, dependsOnEngineType, reachable, latencyMs)
	if err != nil {
		return fmt.Errorf("recording dependency probe: %w", err)
	}
	return nil
}

func (r *Repository) ListDependencies(ctx context.Context, instanceID uuid.UUID) ([]core.InstanceDependency, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT instance_id, depends_on_engine_type, declared, last_probe_reachable, last_probe_latency_ms, last_probed_at
		FROM instance_dependencies WHERE instance_id = $1
	`, instanceID)
	if err != nil {
		return nil, fmt.Errorf("listing dependencies: %w", err)
	}
	defer rows.Close()

	var out []core.InstanceDependency
	for rows.Next() {
		var d core.InstanceDependency
		if err := rows.Scan(&d.InstanceID, &d.DependsOnEngineType, &d.Declared, &d.LastProbeReachable, &d.LastProbeLatencyMs, &d.LastProbedAt); err != nil {
			return nil, fmt.Errorf("scanning dependency: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) GetTopology(ctx context.Context, environmentID *uuid.UUID) (core.TopologyGraph, error) {
	graph := core.TopologyGraph{
		Nodes: []core.TopologyNode{},
		Edges: []core.TopologyEdge{},
	}

	nodeRows, err := r.pool.Query(ctx, `
		SELECT i.id, i.instance_id, i.engine_type, i.environment_id, e.slug, i.status, i.version, i.endpoints
		FROM instances i
		JOIN environments e ON i.environment_id = e.id
		WHERE ($1::uuid IS NULL OR i.environment_id = $1)
		  AND i.status != 'evicted' AND i.status != 'deregistered'
		ORDER BY i.engine_type, i.instance_id
	`, environmentID)
	if err != nil {
		return graph, fmt.Errorf("querying topology nodes: %w", err)
	}
	defer nodeRows.Close()

	for nodeRows.Next() {
		var n core.TopologyNode
		var endpointsJSON []byte
		if err := nodeRows.Scan(&n.ID, &n.InstanceID, &n.EngineType, &n.EnvironmentID, &n.Environment, &n.Status, &n.Version, &endpointsJSON); err != nil {
			return graph, fmt.Errorf("scanning topology node: %w", err)
		}
		if len(endpointsJSON) > 0 {
			_ = json.Unmarshal(endpointsJSON, &n.Endpoints)
		}
		graph.Nodes = append(graph.Nodes, n)
	}
	if err := nodeRows.Err(); err != nil {
		return graph, err
	}

	edgeRows, err := r.pool.Query(ctx, `
		SELECT d.instance_id, i.engine_type, d.depends_on_engine_type, d.declared, d.last_probe_reachable, d.last_probe_latency_ms, d.last_probed_at
		FROM instance_dependencies d
		JOIN instances i ON d.instance_id = i.id
		WHERE ($1::uuid IS NULL OR i.environment_id = $1)
		  AND i.status != 'evicted' AND i.status != 'deregistered'
		ORDER BY i.engine_type, d.depends_on_engine_type
	`, environmentID)
	if err != nil {
		return graph, fmt.Errorf("querying topology edges: %w", err)
	}
	defer edgeRows.Close()

	for edgeRows.Next() {
		var e core.TopologyEdge
		if err := edgeRows.Scan(&e.SourceInstanceID, &e.SourceEngineType, &e.TargetEngineType, &e.Declared, &e.LastProbeReachable, &e.LastProbeLatencyMs, &e.LastProbedAt); err != nil {
			return graph, fmt.Errorf("scanning topology edge: %w", err)
		}
		graph.Edges = append(graph.Edges, e)
	}
	return graph, edgeRows.Err()
}

// ---------------------------------------------------------------------
// Retention (P2-S2-T6)
// ---------------------------------------------------------------------

// PruneOlderThan deletes timeline events older than cutoff, and evicted
// instances not updated since cutoff (which cascades to their credentials
// and any remaining events via ON DELETE CASCADE). Deregistered and
// currently-active instances are never touched here — only an explicit
// force-remove deletes a non-evicted instance.
func (r *Repository) PruneOlderThan(ctx context.Context, cutoff time.Time) (eventsDeleted int64, instancesDeleted int64, err error) {
	evTag, err := r.pool.Exec(ctx, `DELETE FROM instance_events WHERE occurred_at < $1`, cutoff)
	if err != nil {
		return 0, 0, fmt.Errorf("pruning instance events: %w", err)
	}
	instTag, err := r.pool.Exec(ctx, `DELETE FROM instances WHERE status = $1 AND updated_at < $2`, core.StatusEvicted, cutoff)
	if err != nil {
		return evTag.RowsAffected(), 0, fmt.Errorf("pruning evicted instances: %w", err)
	}
	return evTag.RowsAffected(), instTag.RowsAffected(), nil
}

// ---------------------------------------------------------------------
// Enrollment audit trail (P2-S3-T6)
// ---------------------------------------------------------------------

func (r *Repository) AppendEnrollmentAudit(ctx context.Context, entry core.EnrollmentAuditEntry) error {
	detailJSON, err := json.Marshal(entry.Detail)
	if err != nil {
		return fmt.Errorf("marshaling audit detail: %w", err)
	}
	var envID *uuid.UUID
	if entry.EnvironmentID != uuid.Nil {
		envID = &entry.EnvironmentID
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO enrollment_audit_log (token_id, engine_type, environment_id, actor, origin, action, detail)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, entry.TokenID, entry.EngineType, envID, entry.Actor, entry.Origin, entry.Action, detailJSON)
	if err != nil {
		return fmt.Errorf("appending enrollment audit entry: %w", err)
	}
	return nil
}

func (r *Repository) ListEnrollmentAudit(ctx context.Context, filter core.EnrollmentAuditFilter) ([]core.EnrollmentAuditEntry, string, bool, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT id, token_id, engine_type, environment_id, actor, origin, action, detail, occurred_at FROM enrollment_audit_log WHERE 1=1`
	var args []interface{}
	arg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if filter.TokenID != uuid.Nil {
		query += " AND token_id = " + arg(filter.TokenID)
	}
	if filter.Action != "" {
		query += " AND action = " + arg(filter.Action)
	}
	if filter.Cursor != "" {
		var cursorID int64
		if _, err := fmt.Sscanf(filter.Cursor, "%d", &cursorID); err != nil {
			return nil, "", false, fmt.Errorf("parsing cursor: %w", err)
		}
		query += " AND id < " + arg(cursorID)
	}
	query += " ORDER BY id DESC LIMIT " + arg(limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", false, fmt.Errorf("listing enrollment audit: %w", err)
	}
	defer rows.Close()

	var out []core.EnrollmentAuditEntry
	for rows.Next() {
		var e core.EnrollmentAuditEntry
		var tokenID *uuid.UUID
		var envID *uuid.UUID
		var detail []byte
		if err := rows.Scan(&e.ID, &tokenID, &e.EngineType, &envID, &e.Actor, &e.Origin, &e.Action, &detail, &e.OccurredAt); err != nil {
			return nil, "", false, fmt.Errorf("scanning enrollment audit entry: %w", err)
		}
		e.TokenID = tokenID
		if envID != nil {
			e.EnvironmentID = *envID
		}
		if len(detail) > 0 {
			if err := json.Unmarshal(detail, &e.Detail); err != nil {
				return nil, "", false, fmt.Errorf("unmarshaling audit detail: %w", err)
			}
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, "", false, err
	}

	hasMore := len(out) > limit
	if hasMore {
		out = out[:limit]
	}
	nextCursor := ""
	if hasMore && len(out) > 0 {
		nextCursor = fmt.Sprintf("%d", out[len(out)-1].ID)
	}
	return out, nextCursor, hasMore, nil
}

// ---------------------------------------------------------------------
// Console Identity: Operators & Credentials (P3-S1-T1)
// ---------------------------------------------------------------------

func (r *Repository) CountOperators(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM operators`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting operators: %w", err)
	}
	return count, nil
}

func (r *Repository) CreateOperatorWithPassword(ctx context.Context, op core.Operator, passwordHash string) (core.Operator, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return core.Operator{}, fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var created core.Operator
	err = tx.QueryRow(ctx, `
		INSERT INTO operators (email, name, role, is_local, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, name, role, is_local, is_active, created_at, updated_at
	`, op.Email, op.Name, op.Role, op.IsLocal, op.IsActive).Scan(
		&created.ID, &created.Email, &created.Name, &created.Role, &created.IsLocal, &created.IsActive, &created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		return core.Operator{}, fmt.Errorf("inserting operator: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO operator_credentials (operator_id, password_hash)
		VALUES ($1, $2)
	`, created.ID, passwordHash)
	if err != nil {
		return core.Operator{}, fmt.Errorf("inserting operator credentials: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return core.Operator{}, fmt.Errorf("committing operator creation: %w", err)
	}
	return created, nil
}

func (r *Repository) GetOperatorByEmail(ctx context.Context, email string) (core.OperatorWithCredential, error) {
	var op core.OperatorWithCredential
	err := r.pool.QueryRow(ctx, `
		SELECT o.id, o.email, o.name, o.role, o.is_local, o.is_active, o.created_at, o.updated_at,
		       c.password_hash, c.totp_secret, c.totp_enabled, c.failed_attempts, c.locked_until
		FROM operators o
		JOIN operator_credentials c ON o.id = c.operator_id
		WHERE o.email = $1
	`, email).Scan(
		&op.ID, &op.Email, &op.Name, &op.Role, &op.IsLocal, &op.IsActive, &op.CreatedAt, &op.UpdatedAt,
		&op.PasswordHash, &op.TOTPSecret, &op.TOTPEnabled, &op.FailedAttempts, &op.LockedUntil,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return op, core.ErrNotFound
		}
		return op, fmt.Errorf("querying operator by email: %w", err)
	}
	return op, nil
}

func (r *Repository) GetOperatorByID(ctx context.Context, id uuid.UUID) (core.Operator, error) {
	var op core.Operator
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, name, role, is_local, is_active, created_at, updated_at
		FROM operators WHERE id = $1
	`, id).Scan(
		&op.ID, &op.Email, &op.Name, &op.Role, &op.IsLocal, &op.IsActive, &op.CreatedAt, &op.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return op, core.ErrNotFound
		}
		return op, fmt.Errorf("querying operator by id: %w", err)
	}
	return op, nil
}

func (r *Repository) ListOperators(ctx context.Context) ([]core.Operator, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, email, name, role, is_local, is_active, created_at, updated_at
		FROM operators ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing operators: %w", err)
	}
	defer rows.Close()

	var out []core.Operator
	for rows.Next() {
		var op core.Operator
		if err := rows.Scan(&op.ID, &op.Email, &op.Name, &op.Role, &op.IsLocal, &op.IsActive, &op.CreatedAt, &op.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning operator: %w", err)
		}
		out = append(out, op)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateOperatorFailedAttempts(ctx context.Context, id uuid.UUID, failedAttempts int, lockedUntil *time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE operator_credentials
		SET failed_attempts = $2, locked_until = $3, updated_at = now()
		WHERE operator_id = $1
	`, id, failedAttempts, lockedUntil)
	if err != nil {
		return fmt.Errorf("updating operator failed attempts: %w", err)
	}
	return nil
}

func (r *Repository) ResetOperatorFailedAttempts(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE operator_credentials
		SET failed_attempts = 0, locked_until = NULL, updated_at = now()
		WHERE operator_id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("resetting operator failed attempts: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------
// Console Identity: Sessions (P3-S2-T2)
// ---------------------------------------------------------------------

func (r *Repository) CreateSession(ctx context.Context, session core.OperatorSession) (core.OperatorSession, error) {
	var created core.OperatorSession
	err := r.pool.QueryRow(ctx, `
		INSERT INTO operator_sessions (operator_id, token_hash, user_agent, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, operator_id, token_hash, user_agent, ip_address, expires_at, last_active_at, created_at
	`, session.OperatorID, session.TokenHash, session.UserAgent, session.IPAddress, session.ExpiresAt).Scan(
		&created.ID, &created.OperatorID, &created.TokenHash, &created.UserAgent, &created.IPAddress,
		&created.ExpiresAt, &created.LastActiveAt, &created.CreatedAt,
	)
	if err != nil {
		return core.OperatorSession{}, fmt.Errorf("inserting session: %w", err)
	}
	return created, nil
}

func (r *Repository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (core.OperatorSession, core.Operator, error) {
	var s core.OperatorSession
	var o core.Operator
	err := r.pool.QueryRow(ctx, `
		SELECT s.id, s.operator_id, s.token_hash, s.user_agent, s.ip_address, s.expires_at, s.last_active_at, s.created_at,
		       o.id, o.email, o.name, o.role, o.is_local, o.is_active, o.created_at, o.updated_at
		FROM operator_sessions s
		JOIN operators o ON s.operator_id = o.id
		WHERE s.token_hash = $1 AND s.expires_at > now() AND o.is_active = TRUE
	`, tokenHash).Scan(
		&s.ID, &s.OperatorID, &s.TokenHash, &s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.LastActiveAt, &s.CreatedAt,
		&o.ID, &o.Email, &o.Name, &o.Role, &o.IsLocal, &o.IsActive, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s, o, core.ErrNotFound
		}
		return s, o, fmt.Errorf("querying session: %w", err)
	}
	return s, o, nil
}

func (r *Repository) TouchSession(ctx context.Context, sessionID uuid.UUID, lastActiveAt time.Time, extendExpiry time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE operator_sessions
		SET last_active_at = $2, expires_at = $3
		WHERE id = $1
	`, sessionID, lastActiveAt, extendExpiry)
	if err != nil {
		return fmt.Errorf("touching session: %w", err)
	}
	return nil
}

func (r *Repository) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM operator_sessions WHERE id = $1`, sessionID)
	if err != nil {
		return fmt.Errorf("revoking session: %w", err)
	}
	return nil
}

func (r *Repository) RevokeOperatorSessions(ctx context.Context, operatorID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM operator_sessions WHERE operator_id = $1`, operatorID)
	if err != nil {
		return fmt.Errorf("revoking operator sessions: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------
// Console Identity: Bootstrap Tokens (P3-S1-T2)
// ---------------------------------------------------------------------

func (r *Repository) CreateBootstrapToken(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO bootstrap_tokens (token_hash)
		VALUES ($1)
		ON CONFLICT (token_hash) DO NOTHING
	`, tokenHash)
	if err != nil {
		return fmt.Errorf("creating bootstrap token: %w", err)
	}
	return nil
}

func (r *Repository) ConsumeBootstrapToken(ctx context.Context, tokenHash string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE bootstrap_tokens
		SET consumed_at = now()
		WHERE token_hash = $1 AND consumed_at IS NULL
	`, tokenHash)
	if err != nil {
		return false, fmt.Errorf("consuming bootstrap token: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) HasValidBootstrapToken(ctx context.Context) (bool, error) {
	var valid bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM bootstrap_tokens WHERE consumed_at IS NULL
		)
	`).Scan(&valid)
	if err != nil {
		return false, fmt.Errorf("checking valid bootstrap token: %w", err)
	}
	return valid, nil
}

// ---------------------------------------------------------------------
// Console Identity: Role Bindings (P3-S4-T2)
// ---------------------------------------------------------------------

func (r *Repository) CreateRoleBinding(ctx context.Context, rb core.RoleBinding) (core.RoleBinding, error) {
	var created core.RoleBinding
	err := r.pool.QueryRow(ctx, `
		INSERT INTO operator_role_bindings (operator_id, role, environment_id, engine_type)
		VALUES ($1, $2, $3, $4)
		RETURNING id, operator_id, role, environment_id, engine_type, created_at
	`, rb.OperatorID, rb.Role, rb.EnvironmentID, rb.EngineType).Scan(
		&created.ID, &created.OperatorID, &created.Role, &created.EnvironmentID, &created.EngineType, &created.CreatedAt,
	)
	if err != nil {
		return core.RoleBinding{}, fmt.Errorf("inserting role binding: %w", err)
	}
	return created, nil
}

func (r *Repository) ListRoleBindings(ctx context.Context, operatorID uuid.UUID) ([]core.RoleBinding, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, operator_id, role, environment_id, engine_type, created_at
		FROM operator_role_bindings
		WHERE operator_id = $1
		ORDER BY created_at ASC
	`, operatorID)
	if err != nil {
		return nil, fmt.Errorf("listing role bindings: %w", err)
	}
	defer rows.Close()

	var out []core.RoleBinding
	for rows.Next() {
		var rb core.RoleBinding
		if err := rows.Scan(&rb.ID, &rb.OperatorID, &rb.Role, &rb.EnvironmentID, &rb.EngineType, &rb.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning role binding: %w", err)
		}
		out = append(out, rb)
	}
	return out, rows.Err()
}

func (r *Repository) DeleteRoleBinding(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM operator_role_bindings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting role binding: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------
// Console Identity: Personal Access Tokens (P3-S6-T2)
// ---------------------------------------------------------------------

func (r *Repository) CreatePersonalAccessToken(ctx context.Context, pat core.PersonalAccessToken) (core.PersonalAccessToken, error) {
	var created core.PersonalAccessToken
	err := r.pool.QueryRow(ctx, `
		INSERT INTO personal_access_tokens (operator_id, name, token_hash, scopes, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, operator_id, name, token_hash, scopes, expires_at, last_used_at, created_at
	`, pat.OperatorID, pat.Name, pat.TokenHash, pat.Scopes, pat.ExpiresAt).Scan(
		&created.ID, &created.OperatorID, &created.Name, &created.TokenHash, &created.Scopes, &created.ExpiresAt, &created.LastUsedAt, &created.CreatedAt,
	)
	if err != nil {
		return core.PersonalAccessToken{}, fmt.Errorf("inserting personal access token: %w", err)
	}
	return created, nil
}

func (r *Repository) ListPersonalAccessTokens(ctx context.Context, operatorID uuid.UUID) ([]core.PersonalAccessToken, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, operator_id, name, token_hash, scopes, expires_at, last_used_at, created_at
		FROM personal_access_tokens
		WHERE operator_id = $1
		ORDER BY created_at DESC
	`, operatorID)
	if err != nil {
		return nil, fmt.Errorf("listing personal access tokens: %w", err)
	}
	defer rows.Close()

	var out []core.PersonalAccessToken
	for rows.Next() {
		var pat core.PersonalAccessToken
		if err := rows.Scan(&pat.ID, &pat.OperatorID, &pat.Name, &pat.TokenHash, &pat.Scopes, &pat.ExpiresAt, &pat.LastUsedAt, &pat.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning personal access token: %w", err)
		}
		out = append(out, pat)
	}
	return out, rows.Err()
}

func (r *Repository) RevokePersonalAccessToken(ctx context.Context, id uuid.UUID, operatorID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM personal_access_tokens WHERE id = $1 AND operator_id = $2`, id, operatorID)
	if err != nil {
		return fmt.Errorf("revoking personal access token: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------
// Console Identity: Service Accounts (P3-S6-T3)
// ---------------------------------------------------------------------

func (r *Repository) CreateServiceAccount(ctx context.Context, sa core.ServiceAccount) (core.ServiceAccount, error) {
	var created core.ServiceAccount
	err := r.pool.QueryRow(ctx, `
		INSERT INTO service_accounts (name, description, role, token_hash, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, description, role, token_hash, is_active, created_at
	`, sa.Name, sa.Description, sa.Role, sa.TokenHash, sa.IsActive).Scan(
		&created.ID, &created.Name, &created.Description, &created.Role, &created.TokenHash, &created.IsActive, &created.CreatedAt,
	)
	if err != nil {
		return core.ServiceAccount{}, fmt.Errorf("inserting service account: %w", err)
	}
	return created, nil
}

func (r *Repository) ListServiceAccounts(ctx context.Context) ([]core.ServiceAccount, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, role, token_hash, is_active, created_at
		FROM service_accounts
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing service accounts: %w", err)
	}
	defer rows.Close()

	var out []core.ServiceAccount
	for rows.Next() {
		var sa core.ServiceAccount
		if err := rows.Scan(&sa.ID, &sa.Name, &sa.Description, &sa.Role, &sa.TokenHash, &sa.IsActive, &sa.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning service account: %w", err)
		}
		out = append(out, sa)
	}
	return out, rows.Err()
}

func (r *Repository) GetServiceAccountByTokenHash(ctx context.Context, tokenHash string) (core.ServiceAccount, error) {
	var sa core.ServiceAccount
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, role, token_hash, is_active, created_at
		FROM service_accounts
		WHERE token_hash = $1 AND is_active = TRUE
	`, tokenHash).Scan(
		&sa.ID, &sa.Name, &sa.Description, &sa.Role, &sa.TokenHash, &sa.IsActive, &sa.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sa, core.ErrNotFound
		}
		return sa, fmt.Errorf("querying service account: %w", err)
	}
	return sa, nil
}

func (r *Repository) DeleteServiceAccount(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM service_accounts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting service account: %w", err)
	}
	return nil
}


