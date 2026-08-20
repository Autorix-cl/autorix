-- P2-S2-T5: seed the seven known engine types, so the console can render
-- type-aware detail views without hardcoding engine knowledge in the
-- frontend. Ports match each engine's real listeners (ADR 0001 / P1-S1):
-- aegis and nexus advertise their admin/REST port here, never their proxy
-- or gRPC-only port, matching how the health contract itself is exposed.
INSERT INTO engine_types (slug, display_name, default_rest_port, default_grpc_port, protocol, capability_schema) VALUES
    ('nexus',  'Autorix Nexus',  8080, 50051, 'grpc+rest', '{"type": "object", "properties": {"rebac": {"type": "boolean"}, "abac_cel": {"type": "boolean"}}}'::jsonb),
    ('ego',    'Autorix Ego',    4433, NULL,  'rest',      '{"type": "object", "properties": {"argon2id": {"type": "boolean"}}}'::jsonb),
    ('janus',  'Autorix Janus',  4444, NULL,  'rest',      '{"type": "object", "properties": {"pkce": {"type": "boolean"}, "oidc": {"type": "boolean"}}}'::jsonb),
    ('aegis',  'Autorix Aegis',  4456, NULL,  'rest',      '{"type": "object", "properties": {"zero_trust_proxy": {"type": "boolean"}}}'::jsonb),
    ('vulcan', 'Autorix Vulcan', 4466, NULL,  'rest',      '{"type": "object", "properties": {"macaroons": {"type": "boolean"}}}'::jsonb),
    ('hermes', 'Autorix Hermes', 4477, NULL,  'rest',      '{"type": "object", "properties": {"saml": {"type": "boolean"}, "scim": {"type": "boolean"}}}'::jsonb),
    ('themis', 'Autorix Themis', 4488, 50052, 'grpc+rest', '{"type": "object", "properties": {"abac_cel": {"type": "boolean"}}}'::jsonb);
