package core

import (
	"time"

	"github.com/google/uuid"
)

// APIKey represents a registered machine credential
type APIKey struct {
	ID                 uuid.UUID  `json:"id"`
	KeyPrefix          string     `json:"key_prefix"` // "av_live", "av_test"
	KeyHint            string     `json:"key_hint"`   // last 4-6 chars
	KeyHash            string     `json:"-"`
	RootSignatureKey   string     `json:"-"` // secret for deriving Macaroon signatures
	Name               string     `json:"name"`
	OwnerID            string     `json:"owner_id"`
	Scopes             []string   `json:"scopes"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
	State              string     `json:"state"` // "active", "revoked", "expired"
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// Caveat represents a contextual restriction added to a Macaroon
type Caveat struct {
	Predicate string `json:"predicate"` // e.g. "time_before = 2026-08-17T00:00:00Z", "ip = 192.168.1.1"
}

// Macaroon represents a capability token with chained HMAC signatures
type Macaroon struct {
	Location  string   `json:"location"`   // service identifier (e.g. "https://api.autorix.io")
	KeyID     string   `json:"key_id"`     // UUID of the APIKey
	Caveats   []Caveat `json:"caveats"`    // Array of first-party caveats
	Signature string   `json:"signature"` // Hex-encoded final HMAC-SHA256 signature
}

// VerificationContext represents the environment of an incoming request
type VerificationContext struct {
	Now       time.Time `json:"now"`
	IPAddress string    `json:"ip_address"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
}

// CreateKeyRequest payload for issuing a new API key
type CreateKeyRequest struct {
	Name      string     `json:"name"`
	OwnerID   string     `json:"owner_id"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	IsLive    bool       `json:"is_live"` // true -> av_live_, false -> av_test_
}

// CreateKeyResponse returns the raw plaintext token (only shown once)
type CreateKeyResponse struct {
	APIKey   *APIKey   `json:"api_key"`
	RawToken string    `json:"raw_token"` // "av_live_..."
	Macaroon *Macaroon `json:"macaroon"`  // Initial capability token
}
