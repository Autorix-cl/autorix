-- Argus registry schema (P2-S2-T1). Designed so history survives eviction:
-- knowing an engine used to exist and when it disappeared is as
-- operationally valuable as knowing what runs now, so instances and events
-- are never hard-deleted by normal lifecycle transitions — only a
-- configured retention prune (P2-S2-T6) or an explicit force-remove does.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE environments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    is_production BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Every engine self-registers against one of these — a fixed catalog, not
-- an open string, so the console can render type-aware detail views
-- without hardcoding engine knowledge in the frontend (P2-S2-T5 seeds it).
CREATE TABLE engine_types (
    slug TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    default_rest_port INTEGER,
    default_grpc_port INTEGER,
    protocol TEXT NOT NULL,
    capability_schema JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE enrollment_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash TEXT NOT NULL UNIQUE,
    engine_type TEXT NOT NULL REFERENCES engine_types (slug),
    environment_id UUID NOT NULL REFERENCES environments (id),
    uses_allowed INTEGER NOT NULL DEFAULT 1,
    uses_consumed INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_enrollment_tokens_scope ON enrollment_tokens (engine_type, environment_id);

CREATE TABLE instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    engine_type TEXT NOT NULL REFERENCES engine_types (slug),
    instance_id TEXT NOT NULL,
    environment_id UUID NOT NULL REFERENCES environments (id),
    version TEXT NOT NULL DEFAULT '',
    build_sha TEXT NOT NULL DEFAULT '',
    schema_version TEXT NOT NULL DEFAULT '',
    endpoints JSONB NOT NULL DEFAULT '{}'::jsonb,
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    -- pending | registering | healthy | degraded | unreachable | evicted | deregistered
    status TEXT NOT NULL DEFAULT 'pending',
    -- Set once and never overwritten by a later upsert, so "how long has
    -- this instance identity existed" survives restarts.
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_heartbeat_at TIMESTAMPTZ,
    deregistered_at TIMESTAMPTZ,
    enrolled_by TEXT NOT NULL DEFAULT '',
    -- unverified: registered via the manual fallback (P2-S3-T7), not a
    -- minted enrollment token — the trust difference must stay visible.
    unverified BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- The upsert key (P2-S2-T3): a restarting engine updates this row
    -- instead of accumulating duplicates.
    UNIQUE (engine_type, instance_id, environment_id)
);

CREATE INDEX idx_instances_fleet_query ON instances (engine_type, environment_id, status);
CREATE INDEX idx_instances_heartbeat ON instances (last_heartbeat_at);

CREATE TABLE instance_credentials (
    instance_id UUID PRIMARY KEY REFERENCES instances (id) ON DELETE CASCADE,
    secret_hash TEXT NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    rotated_at TIMESTAMPTZ,
    -- Set by force-remove and by rotation once the overlap window closes,
    -- so a zombie process cannot re-authenticate under a revoked secret.
    revoked_at TIMESTAMPTZ,
    -- During a rotation overlap window, the previous secret's hash stays
    -- valid alongside secret_hash until this instant (P2-S3-T3).
    previous_secret_hash TEXT,
    previous_secret_valid_until TIMESTAMPTZ
);

-- Append-only fleet timeline (P2-S2-T4): written in the same transaction
-- as every state transition, so it can never disagree with current state.
CREATE TABLE instance_events (
    id BIGSERIAL PRIMARY KEY,
    instance_id UUID NOT NULL REFERENCES instances (id) ON DELETE CASCADE,
    -- registered | upgraded | degraded | unreachable | deregistered |
    -- evicted | config_changed | flapping
    event_type TEXT NOT NULL,
    detail JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_instance_events_instance ON instance_events (instance_id, occurred_at DESC);
CREATE INDEX idx_instance_events_occurred ON instance_events (occurred_at DESC);

-- Declared (from config) and observed (from heartbeat dependency probes)
-- edges between instances, so the topology graph (P2-S6-T3) can mark an
-- edge that is configured but failing.
CREATE TABLE instance_dependencies (
    id BIGSERIAL PRIMARY KEY,
    instance_id UUID NOT NULL REFERENCES instances (id) ON DELETE CASCADE,
    depends_on_engine_type TEXT NOT NULL REFERENCES engine_types (slug),
    declared BOOLEAN NOT NULL DEFAULT TRUE,
    last_probe_reachable BOOLEAN,
    last_probe_latency_ms DOUBLE PRECISION,
    last_probed_at TIMESTAMPTZ,
    UNIQUE (instance_id, depends_on_engine_type)
);

CREATE INDEX idx_instance_dependencies_instance ON instance_dependencies (instance_id);
