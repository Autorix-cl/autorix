# Integrating and Architecting the Autorix Zero-Trust Ecosystem

This guide explains how the 9 Autorix microservices communicate to enforce zero-trust identity and access management. It serves as your master map for integrating the ecosystem, tracing edge requests down to backend services and audit trails.

## Quick Path: The End-to-End Edge Request Pipeline

When a client makes a request, it flows through the Autorix ecosystem in a decoupled, verifiable pipeline:

1. **Client Request**: Client sends a request to `https://api.enterprise.corp/resource` (handled by **Aegis PEP** on `:4455`).
2. **Authentication**: **Aegis** validates the caller using **Janus** (JWT) or **Vulcan** (Macaroon API keys).
3. **Authorization**: **Aegis** checks permissions in milliseconds against **Nexus** (ReBAC) or **Themis** (ABAC).
4. **Header Mutation**: **Aegis** strips sensitive tokens and injects trusted headers (e.g., `X-User-ID`), forwarding traffic upstream.
5. **Audit Trail**: The upstream service or Aegis records the action to **Argus**, appending it to the SHA-256 Merkle chain.

---

## Details

### 1. Architecture & Port Matrix

The suite consists of decoupled microservices communicating over HTTP REST and gRPC.

| Service | Port(s) | Primary Responsibility | Dedicated Manual |
| :--- | :--- | :--- | :--- |
| **Aegis** | `4455` (Proxy), `4456` (Admin) | Edge PEP, header mutation, routing. | [Aegis Manual](./aegis_usage_guide.md) |
| **Console** | `3000` | Zero-Trust Admin UI, live audit verifier. | [Console Manual](./console_usage_guide.md) |
| **Argus** | `4400` (REST), `50053` (gRPC)| Fleet control, SHA-256 audit trail, compliance. | [Argus Manual](./argus_usage_guide.md) |
| **Nexus** | `8080` (REST), `50051` (gRPC)| Zanzibar ReBAC graph traversal. | [Nexus Manual](./nexus_usage_guide.md) |
| **Themis** | `4488` (REST), `50052` (gRPC)| Contextual CEL-based ABAC. | [Themis Manual](./themis_usage_guide.md) |
| **Ego** | `4433` | Identity lifecycle, dynamic traits, passwords. | [Ego Manual](./ego_usage_guide.md) |
| **Janus** | `4444` | OAuth 2.0 / OIDC tokens, JWKS. | [Janus Manual](./janus_usage_guide.md) |
| **Vulcan** | `4466` | Prefixed API keys, offline Macaroons. | [Vulcan Manual](./vulcan_usage_guide.md) |
| **Hermes** | `4477` | Enterprise SAML 2.0 & SCIM 2.0 bridge. | [Hermes Manual](./hermes_usage_guide.md) |

### 2. Credential & Token Lifecycle Standards

All credentials use recognizable prefixes for easy observability and secret scanning:

*   `abt_`: Bootstrap Token. One-time initial claim of the Argus root master.
*   `aet_`: Enrollment Token. Authorizes new instances to join the cluster.
*   `ast_`: Session Token. Operator session token for Argus/Console.
*   `av_live_`: API Key (Live). Production Macaroon capability tokens.
*   `av_test_`: API Key (Test). Non-production integration testing tokens.
*   `eyJ...`: Access Token. OAuth 2.0 JWT issued by Janus.

### 3. Enterprise Identity Sync Flow

When an enterprise IdP (e.g., Okta) syncs a user:
1. IdP pushes a SCIM 2.0 webhook to **Hermes**.
2. **Hermes** upserts the identity in **Ego**.
3. Group memberships are mapped directly to **Nexus** relation tuples (`group:eng#member@user:<id>`).

### 4. API Conventions: Errors & Pagination

All REST APIs use standard JSON envelopes:
- **Errors**: `{"error": "<type>", "error_description": "<msg>"}`
- **Pagination**: Use `?limit=50&cursor=<opaque_token>`. Response includes `{"data": [...], "next_cursor": "...", "has_more": true}`.

---

## Checklist: Integration Validation

- [ ] Ensure all inter-service gRPC/REST ports (`50051`, `50052`, `8080`, etc.) are open internally but closed to edge traffic.
- [ ] Verify secret scanners are configured to detect `aet_`, `ast_`, and `av_live_` prefixes.
- [ ] Confirm upstream services parse `X-User-ID` headers instead of processing JWTs directly.
- [ ] Ensure Prometheus can scrape `:9090` metrics across all 9 instances.

---

## Next Step

Dive into the specific enforcement engines to configure your security mesh. Start with [Aegis Usage Guide](./aegis_usage_guide.md) for routing or [Argus Usage Guide](./argus_usage_guide.md) for fleet orchestration.
