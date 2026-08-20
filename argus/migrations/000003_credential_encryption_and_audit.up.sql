-- P2-S3-T4: instance credentials must be recoverable server-side to verify
-- an HMAC heartbeat signature, which a one-way hash cannot support — so an
-- AES-256-GCM sealed copy of the raw secret is stored alongside the
-- existing hash (still used for direct-comparison auth, e.g.
-- RotateCredential's current_credential check).
ALTER TABLE instance_credentials
    ADD COLUMN secret_encrypted BYTEA,
    ADD COLUMN previous_secret_encrypted BYTEA;

-- P2-S3-T6: enrollment audit trail — every mint, use, failed attempt and
-- revocation of an enrollment token, independent of instance_events (which
-- is scoped to instances, not tokens, and a failed/rejected enrollment
-- attempt has no instance yet to attach an event to).
CREATE TABLE enrollment_audit_log (
    id BIGSERIAL PRIMARY KEY,
    token_id UUID REFERENCES enrollment_tokens (id) ON DELETE SET NULL,
    engine_type TEXT NOT NULL DEFAULT '',
    environment_id UUID,
    actor TEXT NOT NULL DEFAULT '',
    origin TEXT NOT NULL DEFAULT '',
    -- mint | consume | consume_failed | revoke
    action TEXT NOT NULL,
    detail JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_enrollment_audit_log_token ON enrollment_audit_log (token_id, occurred_at DESC);
CREATE INDEX idx_enrollment_audit_log_occurred ON enrollment_audit_log (occurred_at DESC);
