-- Governance, Audit Trail, and Tenancy Hierarchy (Phase 8: P8-S1, P8-S2, P8-S3, P8-S4)

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 1. Immutable Audit Records with SHA-256 Hash Chaining
CREATE TABLE IF NOT EXISTS audit_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id VARCHAR(255) NOT NULL DEFAULT '',
    actor_type VARCHAR(64) NOT NULL DEFAULT '',
    action VARCHAR(128) NOT NULL,
    resource_type VARCHAR(128) NOT NULL,
    resource_id VARCHAR(255) NOT NULL DEFAULT '',
    environment VARCHAR(128) NOT NULL DEFAULT '',
    before_state JSONB,
    after_state JSONB,
    request_id VARCHAR(255) NOT NULL DEFAULT '',
    source_ip VARCHAR(128) NOT NULL DEFAULT '',
    user_agent VARCHAR(512) NOT NULL DEFAULT '',
    outcome VARCHAR(64) NOT NULL DEFAULT 'success',
    prev_hash VARCHAR(128) NOT NULL DEFAULT '',
    record_hash VARCHAR(128) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_records_resource ON audit_records (resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_records_created_at ON audit_records (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_records_actor ON audit_records (actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_records_action ON audit_records (action);
CREATE INDEX IF NOT EXISTS idx_audit_records_environment ON audit_records (environment);

-- 2. Configuration Revision History and Rollback
CREATE TABLE IF NOT EXISTS config_revisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    engine VARCHAR(128) NOT NULL DEFAULT '',
    resource_type VARCHAR(128) NOT NULL DEFAULT '',
    resource_id VARCHAR(255) NOT NULL DEFAULT '',
    revision_num INT NOT NULL DEFAULT 1,
    author VARCHAR(255) NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_config_revisions_resource ON config_revisions (engine, resource_type, resource_id, revision_num DESC);

-- 3. Multi-Tenant Hierarchy (Organisations -> Projects -> Environments)
CREATE TABLE IF NOT EXISTS organisations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_organisations_slug ON organisations (slug);

CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_projects_org ON projects (org_id);

CREATE TABLE IF NOT EXISTS environments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(64) NOT NULL DEFAULT 'development',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE environments ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE CASCADE;
ALTER TABLE environments ADD COLUMN IF NOT EXISTS type VARCHAR(64) NOT NULL DEFAULT 'development';

CREATE INDEX IF NOT EXISTS idx_environments_project ON environments (project_id);

-- 4. Change Requests (Four-Eyes Principle / Approval Workflow)
CREATE TABLE IF NOT EXISTS change_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_id VARCHAR(255) NOT NULL,
    approver_id VARCHAR(255),
    action VARCHAR(128) NOT NULL,
    target_resource VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(64) NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_change_requests_status ON change_requests (status);
CREATE INDEX IF NOT EXISTS idx_change_requests_requester ON change_requests (requester_id);

-- 5. Maintenance Windows (Blackouts / Change Freezes)
CREATE TABLE IF NOT EXISTS maintenance_windows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_maintenance_windows_range ON maintenance_windows (starts_at, ends_at);
