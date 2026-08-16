-- Enable pgcrypto
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. OAuth2 Registered Clients
CREATE TABLE oauth2_clients (
    id VARCHAR(128) PRIMARY KEY, -- client_id
    client_name VARCHAR(255) NOT NULL,
    client_secret_hash VARCHAR(255),
    grant_types TEXT[] NOT NULL DEFAULT '{"authorization_code", "client_credentials", "refresh_token"}',
    response_types TEXT[] NOT NULL DEFAULT '{"code", "token"}',
    redirect_uris TEXT[] NOT NULL DEFAULT '{}',
    scopes TEXT[] NOT NULL DEFAULT '{"openid", "offline_access"}',
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. JWKS Asymmetric Key Storage
CREATE TABLE jwks_keys (
    kid VARCHAR(64) PRIMARY KEY,
    algorithm VARCHAR(32) NOT NULL DEFAULT 'RS256',
    use VARCHAR(32) NOT NULL DEFAULT 'sig',
    private_key_pem TEXT NOT NULL,
    public_key_pem TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. Authorization Grants (Auth Codes with PKCE)
CREATE TABLE oauth2_grants (
    code_hash VARCHAR(128) PRIMARY KEY,
    client_id VARCHAR(128) NOT NULL REFERENCES oauth2_clients(id) ON DELETE CASCADE,
    subject VARCHAR(255) NOT NULL,
    scopes TEXT[] NOT NULL,
    redirect_uri TEXT NOT NULL,
    code_challenge VARCHAR(128),
    code_challenge_method VARCHAR(16),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    consumed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. Issued Tokens and Revocations
CREATE TABLE oauth2_tokens (
    token_hash VARCHAR(128) PRIMARY KEY,
    client_id VARCHAR(128) NOT NULL REFERENCES oauth2_clients(id) ON DELETE CASCADE,
    subject VARCHAR(255) NOT NULL,
    token_type VARCHAR(32) NOT NULL, -- 'refresh_token', 'access_token'
    scopes TEXT[] NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tokens_client ON oauth2_tokens(client_id);
CREATE INDEX idx_tokens_subject ON oauth2_tokens(subject);
