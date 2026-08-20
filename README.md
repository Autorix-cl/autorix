# Autorix: Next-Generation Zero-Trust IAM Ecosystem

**Autorix** is a high-performance, modular Zero-Trust Identity and Access Management (IAM) suite engineered in Go and TypeScript/Next.js. Inspired by the Ory architecture and Google Zanzibar, Autorix delivers decoupled microservices providing authentication, ReBAC/ABAC authorization, OAuth2/OIDC, reverse proxy PEP enforcement, high-entropy Macaroon API keys, enterprise SAML/SCIM directory federation, cryptographic immutable audit logging, and continuous SOC 2/ISO 27001 compliance verification.

---

## 🏛️ Ecosystem Architecture

```text
                                         [ EXTERNAL TRAFFIC ]
                                                   │
                                                   ▼
                                        ┌─────────────────────┐
                                        │    Autorix Aegis    │  (Zero Trust PEP Reverse Proxy :4455)
                                        └──────────┬──────────┘
                                                   │
        ┌──────────────────────┬───────────────────┼───────────────────┬──────────────────────┐
        ▼                      ▼                   ▼                   ▼                      ▼
 ┌──────────────┐       ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       ┌──────────────┐
 │Autorix Janus │       │Autorix Nexus │    │ Autorix Ego  │    │Autorix Vulcan│       │Autorix Hermes│
 │(OAuth2/OIDC) │       │(ReBAC/Zanzibar)   │(Identity/MFA)│    │ (API Keys)   │       │ (SAML/SCIM)  │
 │    :4444     │       │:50051 / :8080│    │    :4433     │    │    :4466     │       │    :4477     │
 └──────────────┘       └──────────────┘    └──────────────┘    └──────────────┘       └──────────────┘
        │                      │                   │                   │                      │
        └──────────────────────┼───────────────────┼───────────────────┼──────────────────────┘
                               │                   │                   │
                               ▼                   ▼                   ▼
                        ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
                        │Autorix Themis│    │Autorix Argus │    │Autorix Console
                        │ (ABAC / CEL) │    │(Control Plane│    │(Next.js 15 UI│
                        │:50052 / :4488│    │:50053 / :4400│    │    :3000     │
                        └──────────────┘    └──────────────┘    └──────────────┘
```

### Microservice Catalog

| Component | Ports | Protocol | Role & Architecture |
| :--- | :--- | :--- | :--- |
| **Autorix Argus** | `4400` (HTTP) / `50053` (gRPC) | REST / gRPC | **Fleet Control Plane & Governance**: Token enrollment (`aet_...`), instance registry, heartbeats, topology, operator sessions, SHA-256 chained audit trail, continuous compliance. |
| **Autorix Console** | `3000` | HTTP / React | **Administrative Control Plane UI**: Next.js 15 App Router, isolated `/login` & `/setup` shells, Studio management interfaces, live Merkle audit verification, real-time fleet notifications. |
| **Autorix Nexus** | `50051` (gRPC) / `8080` (HTTP) | gRPC / REST | **Fine-Grained ReBAC Engine**: Google Zanzibar relation tuples (`user:X#member@group:Y`), graph traversal, high-throughput Check API. |
| **Autorix Ego** | `4433` | REST | **Identity & Lifecycle Engine**: Dynamic JSON schema identity traits, Argon2id password hashing, TOTP MFA, recovery codes, secure sessions. |
| **Autorix Janus** | `4444` | REST / OIDC | **OAuth 2.0 & OIDC Server**: PKCE S256, RS256 JWKS asymmetric keys, authorization code, client credentials, refresh token flows. |
| **Autorix Aegis** | `4455` (Proxy) / `4456` (Admin) | HTTP Reverse PEP | **Zero-Trust Access Proxy**: Intercepts requests, enforces Bearer/API-key authenticators, evaluates Nexus/Themis authorizers, mutates upstream headers. |
| **Autorix Vulcan** | `4466` | REST | **Capability & API Key Engine**: High-entropy prefixed keys (`av_live_...`), HMAC-SHA256 chained Macaroons with offline caveat attenuation. |
| **Autorix Hermes** | `4477` | REST / XML | **Enterprise Federation Bridge**: SAML 2.0 SP/IdP bridge, SCIM 2.0 (RFC 7643/RFC 7644) automated user/group directory synchronization. |
| **Autorix Themis** | `50052` (gRPC) / `4488` (HTTP) | gRPC / REST | **ABAC Policy Engine**: Google Common Expression Language (CEL) policy evaluator with priority ranking and context injection. |
| **Prometheus** | `9090` | HTTP | **Telemetry & Observability**: Standard RED metrics (Rate, Errors, Duration) across all 7 engines and Argus control plane. |

---

## 🚀 Quick Start (Docker Compose)

Launch the entire 9-service cluster with isolated PostgreSQL databases and Prometheus monitoring with a single command:

```bash
docker compose up -d --build
```

### Initial Bootstrap & Root Owner Setup

1. Open your browser and navigate to the Console:
   ```text
   http://localhost:3000
   ```
2. **First-Time Setup**: If no root administrator exists, Autorix automatically redirects you to the Initial Bootstrap Wizard (`/setup`).
3. Enter the bootstrap token generated by Argus (visible in Docker logs or configured during deployment):
   ```bash
   docker logs autorix-argus | grep "Bootstrap token generated"
   ```
4. Create the Master Administrator account. Once created, the cluster is locked and ready for operation.

### Default Local Development Credentials

| Role | Email | Password |
| :--- | :--- | :--- |
| **Master Owner** | `admin@autorix.local` | `SecretMasterKey#2026` |

---

## 🧪 Testing & Validation

All microservices adhere to strict test coverage requirements (unit tests, integration testcontainers, and Playwright E2E suites):

```bash
# Run backend tests
(cd argus && go test -v ./...)
(cd nexus && go test -v ./...)
(cd ego && go test -v ./...)
(cd janus && go test -v ./...)
(cd aegis && go test -v ./...)
(cd vulcan && go test -v ./...)
(cd hermes && go test -v ./...)
(cd themis && go test -v ./...)

# Run console unit tests and production build
(cd console && npm test && npm run build)

# Run console Playwright E2E acceptance tests
(cd console && npx playwright test)
```

---

## 📚 Documentation & User Manuals

* 📘 [Master API Reference & Integration Guide](docs/api_reference_and_integration_guide.md)
* 🎯 [Autorix Argus: Control Plane, Audit Trail & Compliance Guide](docs/argus_usage_guide.md)
* 🖥️ [Autorix Console: Administration & Studio User Guide](docs/console_usage_guide.md)
* ⚖️ [Autorix Themis: ABAC CEL Policy Engine Guide](docs/themis_usage_guide.md)
* 🧠 [Autorix Nexus: Zanzibar ReBAC & Authorization Guide](docs/nexus_usage_guide.md)
* 👤 [Autorix Ego: Identity, Argon2id & MFA Guide](docs/ego_usage_guide.md)
* 🔑 [Autorix Janus: OAuth2 & OpenID Connect Server Guide](docs/janus_usage_guide.md)
* 🛡️ [Autorix Aegis: Zero-Trust Reverse PEP Proxy Guide](docs/aegis_usage_guide.md)
* ⚡ [Autorix Vulcan: Macaroons & API Key Management Guide](docs/vulcan_usage_guide.md)
* 🏢 [Autorix Hermes: SAML 2.0 & SCIM 2.0 Federation Guide](docs/hermes_usage_guide.md)
* 🛠️ [Operations, Deployment & Troubleshooting Runbook](docs/operations_and_runbook.md)
* ☸️ [Production Kubernetes Deployment Guide](docs/production_k8s_guide.md)
* 📦 [SDKs & Client Libraries Integration Guide](docs/sdks_integration_guide.md)

---

## 🔒 Security Principles

* **Zero Standing Privileges**: Access is checked continuously at ingress (Aegis PEP) and evaluated at the engine level (Nexus/Themis).
* **Never Fake a Signal**: All telemetry, latency figures, audit trails, and compliance controls are backed by real database records and cryptographic proofs.
* **Tamper-Evident Hash Chains**: Every administrative modification in Argus creates a SHA-256 linked audit record verifiable via `/v1/audit/verify`.
* **Memory-Hard Cryptography**: Passwords enforce Argon2id (64MB memory cost, 3 iterations).
* **Decentralized Attenuation**: API keys use HMAC-SHA256 Macaroons that can be restricted with caveats by clients without database lookups.
