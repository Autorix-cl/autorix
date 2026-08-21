package core

import (
	"time"

	"github.com/google/uuid"
)

const (
	StateActive    = "active"
	StateInactive  = "inactive"
	StateSuspended = "suspended"
)

// Identity represents a user in the Autorix ecosystem
type Identity struct {
	ID        uuid.UUID              `json:"id"`
	SchemaID  string                 `json:"schema_id"`
	Traits    map[string]interface{} `json:"traits"`
	State     string                 `json:"state"` // "active", "inactive", "suspended"
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	DeletedAt *time.Time             `json:"deleted_at,omitempty"`
}

// UpdateIdentityPayload defines fields that can be updated on an identity
type UpdateIdentityPayload struct {
	SchemaID *string                `json:"schema_id,omitempty"`
	Traits   map[string]interface{} `json:"traits,omitempty"`
	State    *string                `json:"state,omitempty"`
}

// Credential represents an authentication mechanism attached to an identity
type Credential struct {
	ID             uuid.UUID              `json:"id"`
	IdentityID     uuid.UUID              `json:"identity_id"`
	CredentialType string                 `json:"credential_type"` // "password", "totp"
	CredentialData map[string]interface{} `json:"-"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// Session represents an authenticated user session
type Session struct {
	ID              uuid.UUID `json:"id"`
	IdentityID      uuid.UUID `json:"identity_id"`
	Identity        *Identity `json:"identity,omitempty"`
	Token           string    `json:"token,omitempty"` // Only present upon creation
	TokenHash       string    `json:"-"`
	ExpiresAt       time.Time `json:"expires_at"`
	AuthenticatedAt time.Time `json:"authenticated_at"`
}

// IsActive returns whether the session is currently valid
func (s *Session) IsActive() bool {
	return time.Now().Before(s.ExpiresAt)
}

// RegistrationPayload holds data to register a new user
type RegistrationPayload struct {
	Traits   map[string]interface{} `json:"traits"`
	Password string                 `json:"password"`
}

// LoginPayload holds credentials for authentication
type LoginPayload struct {
	Identifier string `json:"identifier"` // Email / Username
	Password   string `json:"password"`
}

// CredentialInspection exposes credential metadata without leaking secret hashes
type CredentialInspection struct {
	ID             uuid.UUID `json:"id"`
	IdentityID     uuid.UUID `json:"identity_id"`
	CredentialType string    `json:"credential_type"` // "password", "totp", "recovery_token"
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ForceRotation  bool      `json:"force_rotation,omitempty"`
}

// ResetPasswordPayload specifies an optional new password and forced rotation flag
type ResetPasswordPayload struct {
	Password      *string `json:"password,omitempty"`
	ForceRotation bool    `json:"force_rotation"`
}

// ResetPasswordResult contains the outcome of an admin password reset
type ResetPasswordResult struct {
	Status            string  `json:"status"`
	TemporaryPassword *string `json:"temporary_password,omitempty"`
	ForceRotation     bool    `json:"force_rotation"`
}

// RecoveryLinkPayload specifies recovery link parameters
type RecoveryLinkPayload struct {
	ExpiresIn string `json:"expires_in,omitempty"`
}

// RecoveryLinkResult returns the generated recovery link and expiry
type RecoveryLinkResult struct {
	RecoveryLink string    `json:"recovery_link"`
	Token        string    `json:"token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// IdentitySchema represents a traits JSON schema definition
type IdentitySchema struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Schema    map[string]interface{} `json:"schema"`
	Version   int                    `json:"version"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// CreateSchemaPayload defines fields to create a new schema
type CreateSchemaPayload struct {
	ID     string                 `json:"id"`
	Name   string                 `json:"name"`
	Schema map[string]interface{} `json:"schema"`
}

// UpdateSchemaPayload defines fields to update an existing schema
type UpdateSchemaPayload struct {
	Name   *string                `json:"name,omitempty"`
	Schema map[string]interface{} `json:"schema,omitempty"`
}



// FlowUINode represents a single UI element (input field, text, etc)
type FlowUINode struct {
	Type       string                 `json:"type"`  // "input", "text", "script"
	Group      string                 `json:"group"` // "default", "password", "webauthn"
	Attributes map[string]interface{} `json:"attributes"`
	Messages   []interface{}          `json:"messages,omitempty"`
	Meta       map[string]interface{}  `json:"meta,omitempty"`
}

// IdentityFlow represents a state-machine flow (e.g. registration, login, settings)
type IdentityFlow struct {
	ID        uuid.UUID    `json:"id"`
	FlowType  string       `json:"type"`  // "registration", "login", etc
	State     string       `json:"state"` // "choose_method", "passed_challenge"
	UINodes   []FlowUINode `json:"ui_nodes"`
	CSRFToken string       `json:"-"`
	ExpiresAt time.Time    `json:"expires_at"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// FlowSubmitPayload is used to submit data to a flow
type FlowSubmitPayload struct {
	Method    string                 `json:"method"` // "password", "webauthn"
	CSRFToken string                 `json:"csrf_token"`
	Traits    map[string]interface{} `json:"traits,omitempty"`
	Password  string                 `json:"password,omitempty"`
}
