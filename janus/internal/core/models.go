package core

import (
	"time"
)

// OAuth2Client represents a registered relying party / client application
type OAuth2Client struct {
	ID               string    `json:"client_id"`
	ClientName       string    `json:"client_name"`
	ClientSecretHash string    `json:"-"`
	GrantTypes       []string  `json:"grant_types"`
	ResponseTypes    []string  `json:"response_types"`
	RedirectURIs     []string  `json:"redirect_uris"`
	Scopes           []string  `json:"scopes"`
	IsPublic         bool      `json:"is_public"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Grant represents an authorization code grant awaiting exchange
type Grant struct {
	CodeHash            string    `json:"-"`
	ClientID            string    `json:"client_id"`
	Subject             string    `json:"sub"` // User ID from Ego
	Scopes              []string  `json:"scopes"`
	RedirectURI         string    `json:"redirect_uri"`
	CodeChallenge       string    `json:"code_challenge"`
	CodeChallengeMethod string    `json:"code_challenge_method"`
	ExpiresAt           time.Time `json:"expires_at"`
	Consumed            bool      `json:"consumed"`
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
