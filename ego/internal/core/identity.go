package core

import (
	"time"

	"github.com/google/uuid"
)

// Identity represents a user in the Autorix ecosystem
type Identity struct {
	ID        uuid.UUID              `json:"id"`
	SchemaID  string                 `json:"schema_id"`
	Traits    map[string]interface{} `json:"traits"`
	State     string                 `json:"state"` // "active", "suspended"
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
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
