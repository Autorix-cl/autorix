---
layout: home

hero:
  name: "Autorix IAM"
  text: "Next-Gen Zero-Trust Identity & Access Suite"
  tagline: "Microservice-native IAM inspired by Ory and Google Zanzibar. Ultra-low latency ReBAC, ABAC CEL, OAuth2/OIDC, Reverse PEP Proxy, Macaroons, and SHA-256 Chained Audit Logging."
  image:
    src: /logo.svg
    alt: Autorix IAM Logo
  actions:
    - theme: brand
      text: Getting Started
      link: /api_reference_and_integration_guide
    - theme: alt
      text: Explore Engines
      link: /nexus_usage_guide
    - theme: alt
      text: Operations Runbook
      link: /operations_and_runbook

features:
  - icon: 🧠
    title: Nexus (Zanzibar ReBAC)
    details: High-throughput relationship-based access control with Google Zanzibar tuple graphs, cycle detection, and sub-5ms Check evaluation.
    link: /nexus_usage_guide

  - icon: ⚖️
    title: Themis (ABAC CEL)
    details: Contextual, attribute-based policy evaluation powered by Google Common Expression Language (CEL) with priority-based rings.
    link: /themis_usage_guide

  - icon: 👤
    title: Ego (Identity & Traits)
    details: Dynamic JSON Schema user attributes, memory-hard Argon2id password hashing, RFC 6238 TOTP MFA, and active session management.
    link: /ego_usage_guide

  - icon: 🔑
    title: Janus (OAuth2 & OIDC)
    details: Full OpenID Connect provider with PKCE S256, automated RS256 JWKS key rotation, and Client Credentials / Auth Code grants.
    link: /janus_usage_guide

  - icon: 🛡️
    title: Aegis (Zero-Trust PEP Proxy)
    details: Reverse access proxy enforcing authentication, Zanzibar/CEL authorization, and header mutation at perimeter ingress.
    link: /aegis_usage_guide

  - icon: ⚡
    title: Vulcan (Macaroon API Keys)
    details: High-entropy prefixed API keys with offline, decentralized HMAC-SHA256 caveat attenuation and constant-time verification.
    link: /vulcan_usage_guide

  - icon: 🏢
    title: Hermes (SAML & SCIM)
    details: Enterprise IdP federation (Okta, Azure AD, Google) via SAML 2.0 Web SSO and automated SCIM 2.0 directory synchronization.
    link: /hermes_usage_guide

  - icon: 🎯
    title: Argus (Fleet Control Plane)
    details: Engine enrollment tokens, continuous health heartbeats, topology visualization, and tamper-evident SHA-256 chained audit trails.
    link: /argus_usage_guide

  - icon: 🖥️
    title: Autorix Console (Next.js 15)
    details: Enterprise management dashboard with dedicated Studio workspaces, live cryptographic audit verification, and real-time alerts.
    link: /console_usage_guide
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
