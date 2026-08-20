-- Enable pgcrypto
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. SAML 2.0 Providers
CREATE TABLE saml_providers (
    id VARCHAR(128) PRIMARY KEY,
    display_name VARCHAR(255) NOT NULL,
    idp_entity_id VARCHAR(512) NOT NULL,
    idp_sso_url VARCHAR(512) NOT NULL,
    idp_certificate_pem TEXT NOT NULL,
    idp_cert_expires_at TIMESTAMP WITH TIME ZONE,
    sp_entity_id VARCHAR(512) NOT NULL,
    attribute_mapping JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. SCIM 2.0 Synced Users (RFC 7643)
CREATE TABLE scim_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id VARCHAR(255) NOT NULL UNIQUE,
    user_name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255),
    email VARCHAR(255) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    attributes JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. SCIM 2.0 Synced Groups (RFC 7643)
CREATE TABLE scim_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name VARCHAR(255) NOT NULL UNIQUE,
    members JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. SCIM Sync History (RFC 7644 sync state tracking)
CREATE TABLE scim_sync_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id VARCHAR(128),
    resource_type VARCHAR(64) NOT NULL DEFAULT 'All',
    status VARCHAR(32) NOT NULL DEFAULT 'success',
    total_records INT NOT NULL DEFAULT 0,
    created_count INT NOT NULL DEFAULT 0,
    updated_count INT NOT NULL DEFAULT 0,
    deleted_count INT NOT NULL DEFAULT 0,
    error_count INT NOT NULL DEFAULT 0,
    errors JSONB NOT NULL DEFAULT '[]',
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_scim_users_email ON scim_users(email);
CREATE INDEX idx_scim_users_external_id ON scim_users(external_id);
CREATE INDEX idx_scim_sync_history_started ON scim_sync_history(started_at DESC);

