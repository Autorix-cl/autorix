-- Operators, Credentials and Sessions for Console Identity (Phase 3: P3-S1, P3-S2)

CREATE TABLE operators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'operator', -- owner | admin | operator | auditor
    is_local BOOLEAN NOT NULL DEFAULT TRUE, -- true = break-glass / local; false = SSO
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_operators_email ON operators (email);
CREATE INDEX idx_operators_role ON operators (role);

CREATE TABLE operator_credentials (
    operator_id UUID PRIMARY KEY REFERENCES operators (id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    totp_secret TEXT,
    totp_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    failed_attempts INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE operator_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operator_id UUID NOT NULL REFERENCES operators (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    user_agent TEXT NOT NULL DEFAULT '',
    ip_address TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_operator_sessions_token_hash ON operator_sessions (token_hash);
CREATE INDEX idx_operator_sessions_operator ON operator_sessions (operator_id);
CREATE INDEX idx_operator_sessions_expires ON operator_sessions (expires_at);

CREATE TABLE bootstrap_tokens (
    token_hash TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    consumed_at TIMESTAMPTZ
);
