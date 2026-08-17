-- Themis: Business Rules Engine - Policy Storage
-- Each policy belongs to a tenant and contains a CEL expression
-- that is evaluated against dynamic JSON payloads at runtime.

CREATE TABLE policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(128) NOT NULL,
    name VARCHAR(256) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    expression TEXT NOT NULL,           -- Google CEL expression
    priority INTEGER NOT NULL DEFAULT 0, -- Lower = higher priority
    enabled BOOLEAN NOT NULL DEFAULT true,
    labels JSONB NOT NULL DEFAULT '{}',  -- Arbitrary key-value metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- A tenant cannot have two policies with the same name
    CONSTRAINT uq_tenant_policy_name UNIQUE (tenant_id, name)
);

-- Index for fast lookup by tenant (most common query pattern)
CREATE INDEX idx_policies_tenant
ON policies (tenant_id);

-- Index for filtering enabled policies per tenant (evaluation hot path)
CREATE INDEX idx_policies_tenant_enabled
ON policies (tenant_id, enabled) WHERE enabled = true;

-- Index for priority-ordered retrieval
CREATE INDEX idx_policies_priority
ON policies (tenant_id, priority ASC);

-- GIN index for label-based filtering
CREATE INDEX idx_policies_labels
ON policies USING GIN (labels);
