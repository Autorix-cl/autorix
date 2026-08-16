package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/autorix/hermes/internal/core"
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

// CreateSAMLProvider registers a new SAML IdP
func (r *Repository) CreateSAMLProvider(ctx context.Context, p *core.SAMLProvider) error {
	now := time.Now()
	mappingJSON, _ := json.Marshal(p.AttributeMapping)

	query := `
		INSERT INTO saml_providers (
			id, display_name, idp_entity_id, idp_sso_url, idp_certificate_pem,
			sp_entity_id, attribute_mapping, enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		p.ID, p.DisplayName, p.IdPEntityID, p.IdPSSOURL, p.IdPCertificatePEM,
		p.SPEntityID, mappingJSON, p.Enabled, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert saml provider: %w", err)
	}

	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

// GetSAMLProvider retrieves a SAML provider by ID
func (r *Repository) GetSAMLProvider(ctx context.Context, id string) (*core.SAMLProvider, error) {
	query := `
		SELECT id, display_name, idp_entity_id, idp_sso_url, idp_certificate_pem,
		       sp_entity_id, attribute_mapping, enabled, created_at, updated_at
		FROM saml_providers
		WHERE id = $1
	`
	var p core.SAMLProvider
	var mappingJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.DisplayName, &p.IdPEntityID, &p.IdPSSOURL, &p.IdPCertificatePEM,
		&p.SPEntityID, &mappingJSON, &p.Enabled, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query saml provider: %w", err)
	}

	_ = json.Unmarshal(mappingJSON, &p.AttributeMapping)
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
