-- 1. Extend api_keys table with description, usage tracking, and rotation grace period columns
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS call_count BIGINT NOT NULL DEFAULT 0;
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS last_source_ip VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS prev_key_hash VARCHAR(128);
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS prev_root_signature_key VARCHAR(128);
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS grace_period_expires_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_api_keys_prev_hash ON api_keys(prev_key_hash);

-- 2. Create api_key_scopes catalogue table
CREATE TABLE IF NOT EXISTS api_key_scopes (
    name VARCHAR(128) PRIMARY KEY,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
