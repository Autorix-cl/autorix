-- Refresh-token rotation metadata. Refresh token values are always stored as
-- SHA-256 hashes; family_id permits reuse detection to revoke the entire chain.
ALTER TABLE oauth2_tokens
    ADD COLUMN family_id UUID,
    ADD COLUMN resource TEXT NOT NULL DEFAULT '',
    ADD COLUMN rotated_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX idx_oauth2_tokens_refresh_family
    ON oauth2_tokens (family_id)
    WHERE token_type = 'refresh_token';
