-- Aegis: Zero-Trust Access Proxy - Rules and Rule Versions Storage

CREATE TABLE IF NOT EXISTS rules (
    id VARCHAR(128) PRIMARY KEY,
    description TEXT NOT NULL DEFAULT '',
    order_idx INTEGER NOT NULL DEFAULT 0,
    match JSONB NOT NULL,
    authenticators JSONB NOT NULL DEFAULT '[]',
    authorizer JSONB NOT NULL DEFAULT '{}',
    mutators JSONB NOT NULL DEFAULT '[]',
    upstream JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rules_order_idx ON rules (order_idx ASC, created_at ASC);

CREATE TABLE IF NOT EXISTS rule_versions (
    version SERIAL PRIMARY KEY,
    description TEXT NOT NULL DEFAULT '',
    rules_snapshot JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
