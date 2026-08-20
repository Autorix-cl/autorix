package core

import (
	"time"

	"github.com/google/uuid"
)

// OperatorRole defines the permission level of a console operator (P3-S4).
type OperatorRole string

const (
	RoleOwner    OperatorRole = "owner"
	RoleAdmin    OperatorRole = "admin"
	RoleOperator OperatorRole = "operator"
	RoleAuditor  OperatorRole = "auditor"
)

// Operator is a console administrator or operator user (P3-S1-T1).
type Operator struct {
	ID        uuid.UUID    `json:"id"`
	Email     string       `json:"email"`
	Name      string       `json:"name"`
	Role      OperatorRole `json:"role"`
	IsLocal   bool         `json:"is_local"`
	IsActive  bool         `json:"is_active"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// OperatorWithCredential contains the operator metadata plus authentication secrets.
type OperatorWithCredential struct {
	Operator
	PasswordHash   string     `json:"-"`
	TOTPSecret     *string    `json:"-"`
	TOTPEnabled    bool       `json:"totp_enabled"`
	FailedAttempts int        `json:"failed_attempts"`
	LockedUntil    *time.Time `json:"locked_until,omitempty"`
}

// OperatorSession is an authenticated server-side session in Argus (P3-S2-T2).
type OperatorSession struct {
	ID           uuid.UUID `json:"id"`
	OperatorID   uuid.UUID `json:"operator_id"`
	TokenHash    string    `json:"-"`
	UserAgent    string    `json:"user_agent"`
	IPAddress    string    `json:"ip_address"`
	ExpiresAt    time.Time `json:"expires_at"`
	LastActiveAt time.Time `json:"last_active_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// SessionTTL constants (P3-S2-T2)
const (
	DefaultSessionIdleTTL     = 30 * time.Minute
	DefaultSessionAbsoluteTTL = 12 * time.Hour
)

// RoleBinding binds an operator to a role within an environment and engine scope (P3-S4-T2).
type RoleBinding struct {
	ID            uuid.UUID     `json:"id"`
	OperatorID    uuid.UUID     `json:"operator_id"`
	Role          OperatorRole  `json:"role"`
	EnvironmentID *uuid.UUID    `json:"environment_id,omitempty"`
	EngineType    *string       `json:"engine_type,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}

// PersonalAccessToken is a scoped programmatic credential for operators (P3-S6-T2).
type PersonalAccessToken struct {
	ID         uuid.UUID  `json:"id"`
	OperatorID uuid.UUID  `json:"operator_id"`
	Name       string     `json:"name"`
	TokenHash  string     `json:"-"`
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ServiceAccount is a non-human identity for CI/automation (P3-S6-T3).
type ServiceAccount struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Role        OperatorRole `json:"role"`
	TokenHash   string       `json:"-"`
	IsActive    bool         `json:"is_active"`
	CreatedAt   time.Time    `json:"created_at"`
}

