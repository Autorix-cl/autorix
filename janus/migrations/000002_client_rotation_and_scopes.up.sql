-- 1. Client Secret Rotation columns
ALTER TABLE oauth2_clients
    ADD COLUMN previous_secret_hash VARCHAR(255),
    ADD COLUMN previous_secret_expires_at TIMESTAMP WITH TIME ZONE;

-- 2. Scope and Claims Catalogue
CREATE TABLE oauth2_scopes (
    name VARCHAR(64) PRIMARY KEY,
    description TEXT NOT NULL DEFAULT '',
    claims JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
