package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/autorix/hermes/internal/core"
	"github.com/autorix/hermes/internal/saml"
	"github.com/autorix/platform/paging"
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

// populateCertDetails parses PEM and populates certificates metadata and expiration info
func populateCertDetails(p *core.SAMLProvider) {
	if p.IdPCertificatePEM != "" {
		certs, expiresAt, warnings, err := saml.ParseCertificatesPEM(p.IdPCertificatePEM)
		if err == nil {
			p.Certificates = certs
			p.Warnings = warnings
			if expiresAt != nil {
				p.IdPCertExpiresAt = expiresAt
			}
		}
	}
}

// CreateSAMLProvider registers a new SAML IdP
func (r *Repository) CreateSAMLProvider(ctx context.Context, p *core.SAMLProvider) error {
	now := time.Now()
	if p.IdPCertificatePEM == "" {
		p.IdPCertificatePEM = "-----BEGIN CERTIFICATE-----\nMIID...AUTORIX...CERT\n-----END CERTIFICATE-----"
	}
	if p.AttributeMapping == nil {
		p.AttributeMapping = map[string]string{}
	}
	mappingJSON, _ := json.Marshal(p.AttributeMapping)

	populateCertDetails(p)

	query := `
		INSERT INTO saml_providers (
			id, display_name, idp_entity_id, idp_sso_url, idp_certificate_pem,
			idp_cert_expires_at, sp_entity_id, attribute_mapping, enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		p.ID, p.DisplayName, p.IdPEntityID, p.IdPSSOURL, p.IdPCertificatePEM,
		p.IdPCertExpiresAt, p.SPEntityID, mappingJSON, p.Enabled, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert saml provider: %w", err)
	}

	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

// UpdateSAMLProvider updates an existing SAML IdP
func (r *Repository) UpdateSAMLProvider(ctx context.Context, p *core.SAMLProvider) error {
	now := time.Now()
	if p.AttributeMapping == nil {
		p.AttributeMapping = map[string]string{}
	}
	mappingJSON, _ := json.Marshal(p.AttributeMapping)

	populateCertDetails(p)

	query := `
		UPDATE saml_providers
		SET display_name = $1, idp_entity_id = $2, idp_sso_url = $3,
		    idp_certificate_pem = $4, idp_cert_expires_at = $5, sp_entity_id = $6,
		    attribute_mapping = $7, enabled = $8, updated_at = $9
		WHERE id = $10
	`
	cmd, err := r.pool.Exec(ctx, query,
		p.DisplayName, p.IdPEntityID, p.IdPSSOURL,
		p.IdPCertificatePEM, p.IdPCertExpiresAt, p.SPEntityID,
		mappingJSON, p.Enabled, now, p.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update saml provider: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}

	p.UpdatedAt = now
	return nil
}

// SetSAMLProviderEnabled enables or disables a SAML IdP
func (r *Repository) SetSAMLProviderEnabled(ctx context.Context, id string, enabled bool) error {
	now := time.Now()
	query := `UPDATE saml_providers SET enabled = $1, updated_at = $2 WHERE id = $3`
	cmd, err := r.pool.Exec(ctx, query, enabled, now, id)
	if err != nil {
		return fmt.Errorf("failed to toggle saml provider enabled: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteSAMLProvider deletes a SAML IdP by ID
func (r *Repository) DeleteSAMLProvider(ctx context.Context, id string) error {
	query := `DELETE FROM saml_providers WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete saml provider: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListSAMLProviders retrieves registered SAML IdPs using keyset pagination
func (r *Repository) ListSAMLProviders(ctx context.Context, limit int, cursor string) ([]core.SAMLProvider, bool, error) {
	query := `
		SELECT id, display_name, idp_entity_id, idp_sso_url, idp_certificate_pem,
		       idp_cert_expires_at, sp_entity_id, attribute_mapping, enabled, created_at, updated_at
		FROM saml_providers`
	args := []interface{}{}

	if cursor != "" {
		createdAt, id, err := decodeSAMLProviderCursor(cursor)
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
		return nil, false, fmt.Errorf("failed to list saml providers: %w", err)
	}
	defer rows.Close()

	var providers []core.SAMLProvider
	for rows.Next() {
		var p core.SAMLProvider
		var mappingJSON []byte
		if err := rows.Scan(
			&p.ID, &p.DisplayName, &p.IdPEntityID, &p.IdPSSOURL, &p.IdPCertificatePEM,
			&p.IdPCertExpiresAt, &p.SPEntityID, &mappingJSON, &p.Enabled, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, false, err
		}
		_ = json.Unmarshal(mappingJSON, &p.AttributeMapping)
		populateCertDetails(&p)
		providers = append(providers, p)
	}

	hasMore := len(providers) > limit
	if hasMore {
		providers = providers[:limit]
	}
	return providers, hasMore, nil
}

// decodeSAMLProviderCursor reverses the "createdAt|id" cursor payload
func decodeSAMLProviderCursor(cursor string) (time.Time, string, error) {
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

// GetSAMLProvider retrieves a SAML provider by ID
func (r *Repository) GetSAMLProvider(ctx context.Context, id string) (*core.SAMLProvider, error) {
	query := `
		SELECT id, display_name, idp_entity_id, idp_sso_url, idp_certificate_pem,
		       idp_cert_expires_at, sp_entity_id, attribute_mapping, enabled, created_at, updated_at
		FROM saml_providers
		WHERE id = $1
	`
	var p core.SAMLProvider
	var mappingJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.DisplayName, &p.IdPEntityID, &p.IdPSSOURL, &p.IdPCertificatePEM,
		&p.IdPCertExpiresAt, &p.SPEntityID, &mappingJSON, &p.Enabled, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query saml provider: %w", err)
	}

	_ = json.Unmarshal(mappingJSON, &p.AttributeMapping)
	populateCertDetails(&p)
	return &p, nil
}

// CreateSCIMUser persists an RFC 7643 user
func (r *Repository) CreateSCIMUser(ctx context.Context, u *core.SCIMUser) error {
	now := time.Now()
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}

	primaryEmail := ""
	if len(u.Emails) > 0 {
		primaryEmail = u.Emails[0].Value
	}

	attrJSON, _ := json.Marshal(u.Attributes)

	query := `
		INSERT INTO scim_users (
			id, external_id, user_name, display_name, email, active, attributes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		u.ID, u.ExternalID, u.UserName, u.DisplayName, primaryEmail, u.Active, attrJSON, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert scim user: %w", err)
	}

	u.Meta.Created = now
	u.Meta.LastModified = now
	return nil
}

// ListSCIMUsers returns all synchronized users
func (r *Repository) ListSCIMUsers(ctx context.Context) ([]core.SCIMUser, error) {
	query := `
		SELECT id, external_id, user_name, display_name, email, active, created_at, updated_at
		FROM scim_users
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list scim users: %w", err)
	}
	defer rows.Close()

	var users []core.SCIMUser
	for rows.Next() {
		var u core.SCIMUser
		var email string
		err := rows.Scan(
			&u.ID, &u.ExternalID, &u.UserName, &u.DisplayName, &email, &u.Active,
			&u.Meta.Created, &u.Meta.LastModified,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		u.Emails = []core.SCIMEmail{{Value: email, Primary: true, Type: "work"}}
		users = append(users, u)
	}

	return users, nil
}

// CreateSCIMGroup persists an RFC 7643 group
func (r *Repository) CreateSCIMGroup(ctx context.Context, g *core.SCIMGroup) error {
	now := time.Now()
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	if g.Members == nil {
		g.Members = []core.SCIMMember{}
	}

	membersJSON, err := json.Marshal(g.Members)
	if err != nil {
		return fmt.Errorf("failed to marshal scim group members: %w", err)
	}

	query := `
		INSERT INTO scim_groups (
			id, display_name, members, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $4)
	`
	_, err = r.pool.Exec(ctx, query, g.ID, g.DisplayName, membersJSON, now)
	if err != nil {
		return fmt.Errorf("failed to insert scim group: %w", err)
	}

	g.Meta.Created = now
	g.Meta.LastModified = now
	return nil
}

// GetSCIMGroup retrieves a SCIM group by ID
func (r *Repository) GetSCIMGroup(ctx context.Context, id uuid.UUID) (*core.SCIMGroup, error) {
	query := `
		SELECT id, display_name, members, created_at, updated_at
		FROM scim_groups
		WHERE id = $1
	`
	var g core.SCIMGroup
	var membersJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&g.ID, &g.DisplayName, &membersJSON, &g.Meta.Created, &g.Meta.LastModified,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query scim group: %w", err)
	}

	_ = json.Unmarshal(membersJSON, &g.Members)
	return &g, nil
}

// ListSCIMGroups returns all synchronized groups
func (r *Repository) ListSCIMGroups(ctx context.Context) ([]core.SCIMGroup, error) {
	query := `
		SELECT id, display_name, members, created_at, updated_at
		FROM scim_groups
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list scim groups: %w", err)
	}
	defer rows.Close()

	var groups []core.SCIMGroup
	for rows.Next() {
		var g core.SCIMGroup
		var membersJSON []byte
		err := rows.Scan(
			&g.ID, &g.DisplayName, &membersJSON, &g.Meta.Created, &g.Meta.LastModified,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scim group: %w", err)
		}
		_ = json.Unmarshal(membersJSON, &g.Members)
		groups = append(groups, g)
	}

	return groups, nil
}

// UpdateSCIMGroup updates an existing SCIM group
func (r *Repository) UpdateSCIMGroup(ctx context.Context, g *core.SCIMGroup) error {
	now := time.Now()
	if g.Members == nil {
		g.Members = []core.SCIMMember{}
	}

	membersJSON, err := json.Marshal(g.Members)
	if err != nil {
		return fmt.Errorf("failed to marshal scim group members: %w", err)
	}

	query := `
		UPDATE scim_groups
		SET display_name = $1, members = $2, updated_at = $3
		WHERE id = $4
	`
	cmd, err := r.pool.Exec(ctx, query, g.DisplayName, membersJSON, now, g.ID)
	if err != nil {
		return fmt.Errorf("failed to update scim group: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}

	g.Meta.LastModified = now
	return nil
}

// DeleteSCIMGroup deletes a SCIM group by ID
func (r *Repository) DeleteSCIMGroup(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM scim_groups WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete scim group: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RecordSCIMSync logs a synchronization run in scim_sync_history
func (r *Repository) RecordSCIMSync(ctx context.Context, s *core.SCIMSyncHistory) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.StartedAt.IsZero() {
		s.StartedAt = time.Now()
	}
	if s.Errors == nil {
		s.Errors = []string{}
	}

	errorsJSON, err := json.Marshal(s.Errors)
	if err != nil {
		return fmt.Errorf("failed to marshal scim sync errors: %w", err)
	}

	query := `
		INSERT INTO scim_sync_history (
			id, provider_id, resource_type, status, total_records,
			created_count, updated_count, deleted_count, error_count,
			errors, started_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err = r.pool.Exec(ctx, query,
		s.ID, s.ProviderID, s.ResourceType, s.Status, s.TotalRecords,
		s.CreatedCount, s.UpdatedCount, s.DeletedCount, s.ErrorCount,
		errorsJSON, s.StartedAt, s.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert scim sync history: %w", err)
	}
	return nil
}

// ListSCIMSyncHistory retrieves the most recent synchronization runs
func (r *Repository) ListSCIMSyncHistory(ctx context.Context, limit int) ([]core.SCIMSyncHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, provider_id, resource_type, status, total_records,
		       created_count, updated_count, deleted_count, error_count,
		       errors, started_at, completed_at
		FROM scim_sync_history
		ORDER BY started_at DESC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list scim sync history: %w", err)
	}
	defer rows.Close()

	var history []core.SCIMSyncHistory
	for rows.Next() {
		var s core.SCIMSyncHistory
		var errorsJSON []byte
		var providerID *string
		err := rows.Scan(
			&s.ID, &providerID, &s.ResourceType, &s.Status, &s.TotalRecords,
			&s.CreatedCount, &s.UpdatedCount, &s.DeletedCount, &s.ErrorCount,
			&errorsJSON, &s.StartedAt, &s.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scim sync history: %w", err)
		}
		if providerID != nil {
			s.ProviderID = *providerID
		}
		_ = json.Unmarshal(errorsJSON, &s.Errors)
		history = append(history, s)
	}

	return history, nil
}

