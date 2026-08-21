package core

import (
	"time"
)

// OAuth2Client represents a registered relying party / client application
type OAuth2Client struct {
	ID                      string     `json:"client_id"`
	ClientName              string     `json:"client_name"`
	ClientSecretHash        string     `json:"-"`
	PreviousSecretHash      string     `json:"-"`
	PreviousSecretExpiresAt *time.Time `json:"previous_secret_expires_at,omitempty"`
	ClientSecret            string     `json:"client_secret,omitempty"`
	GrantTypes              []string   `json:"grant_types"`
	ResponseTypes           []string   `json:"response_types"`
	RedirectURIs            []string   `json:"redirect_uris"`
	Scopes                  []string   `json:"scopes"`
	IsPublic                bool       `json:"is_public"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

// OAuth2Scope represents a scope definition in the scope catalogue
type OAuth2Scope struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Claims      []string  `json:"claims"`
	CreatedAt   time.Time `json:"created_at"`
}

// Grant represents an authorization code grant awaiting exchange
type Grant struct {
	CodeHash            string    `json:"code_hash,omitempty"`
	ClientID            string    `json:"client_id"`
	Subject             string    `json:"sub"` // User ID from Ego
	Scopes              []string  `json:"scopes"`
	RedirectURI         string    `json:"redirect_uri"`
	CodeChallenge       string    `json:"code_challenge,omitempty"`
	CodeChallengeMethod string    `json:"code_challenge_method,omitempty"`
	ExpiresAt           time.Time `json:"expires_at"`
	Consumed            bool      `json:"consumed"`
	CreatedAt           time.Time `json:"created_at"`
}

// TokenRecord represents a persisted refresh or access token
type TokenRecord struct {
	TokenHash string    `json:"-"`
	ClientID  string    `json:"client_id"`
	Subject   string    `json:"sub"`
	TokenType string    `json:"token_type"` // "refresh_token", "access_token"
	Scopes    []string  `json:"scopes"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}

// JWK represents a JSON Web Key (RFC 7517)
type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKS represents a collection of JSON Web Keys
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// TokenResponse represents RFC 6749 Token Endpoint Response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope"`
}
// LoginChallenge represents a decoupled login request
type LoginChallenge struct {
	Challenge           string     `json:"challenge"`
	ClientID            string     `json:"client_id"`
	RedirectURI         string     `json:"redirect_uri"`
	ResponseType        string     `json:"response_type"`
	Scopes              []string   `json:"scopes"`
	State               string     `json:"state"`
	Nonce               string     `json:"nonce"`
	CodeChallenge       string     `json:"code_challenge"`
	CodeChallengeMethod string     `json:"code_challenge_method"`
	Subject             string     `json:"subject"`
	LoginVerifier       string     `json:"login_verifier"`
	HandledAt           *time.Time `json:"handled_at"`
	CreatedAt           time.Time  `json:"created_at"`
}

// ConsentChallenge represents a decoupled consent request
type ConsentChallenge struct {
	Challenge       string     `json:"challenge"`
	LoginChallenge  string     `json:"login_challenge"`
	ClientID        string     `json:"client_id"`
	Subject         string     `json:"subject"`
	RequestedScopes []string   `json:"requested_scopes"`
	GrantedScopes   []string   `json:"granted_scopes"`
	ConsentVerifier string     `json:"consent_verifier"`
	HandledAt       *time.Time `json:"handled_at"`
	CreatedAt       time.Time  `json:"created_at"`
}
