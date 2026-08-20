-- Role bindings, personal access tokens and service accounts (P3-S4-T2, P3-S6-T2, P3-S6-T3)

CREATE TABLE operator_role_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operator_id UUID NOT NULL REFERENCES operators (id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    environment_id UUID REFERENCES environments (id) ON DELETE CASCADE,
    engine_type TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_operator_role_bindings_operator ON operator_role_bindings (operator_id);
CREATE INDEX idx_operator_role_bindings_env ON operator_role_bindings (environment_id);

CREATE TABLE personal_access_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operator_id UUID NOT NULL REFERENCES operators (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_personal_access_tokens_token_hash ON personal_access_tokens (token_hash);
CREATE INDEX idx_personal_access_tokens_operator ON personal_access_tokens (operator_id);

CREATE TABLE service_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT 'operator',
    token_hash TEXT NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_service_accounts_token_hash ON service_accounts (token_hash);
