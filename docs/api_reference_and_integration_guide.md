# Master Ecosystem Architecture & Integration Flows Guide

This document is the authoritative integration manual and architectural reference for the complete **Autorix Zero-Trust IAM Ecosystem**. For individual endpoint-level specifications, request/response bodies, and parameters, refer directly to each engine's dedicated manual linked below.

---

## 🏛️ 1. Architecture & Port Matrix

The Autorix suite consists of 9 decoupled microservices communicating over high-speed HTTP REST and internal gRPC protocols:

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

### Complete Service Matrix

| Service | Port(s) | Protocol | Ingress Tier | Primary Responsibility | Dedicated Manual |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Autorix Aegis** | `4455` (Proxy)<br>`4456` (Admin) | HTTP / Reverse Proxy<br>HTTP REST | **Edge / Public**<br>Internal Admin | Ingress Policy Enforcement Point (PEP), header mutation, routing. | [Aegis Manual](/aegis_usage_guide) |
| **Autorix Console** | `3000` | HTTP / Next.js 15 | **Admin / Public** | Zero-Trust Admin Console, Studio interfaces, live Merkle audit verifier. | [Console Manual](/console_usage_guide) |
| **Autorix Argus** | `4400` (REST)<br>`50053` (gRPC) | HTTP REST<br>gRPC Protobuf | Admin / Internal<br>Cluster Internal | Fleet control plane, token enrollment (`aet_`), SHA-256 audit trail, compliance. | [Argus Manual](/argus_usage_guide) |
| **Autorix Nexus** | `8080` (REST)<br>`50051` (gRPC) | HTTP REST<br>gRPC Protobuf | Internal / Service<br>Cluster Internal | Google Zanzibar relation graph traversal (ReBAC) and CEL caveats. | [Nexus Manual](/nexus_usage_guide) |
| **Autorix Themis**| `4488` (REST)<br>`50052` (gRPC) | HTTP REST<br>gRPC Protobuf | Internal / Service<br>Cluster Internal | Contextual Attribute-Based Access Control (ABAC) with Google CEL. | [Themis Manual](/themis_usage_guide) |
| **Autorix Ego** | `4433` | HTTP REST | Public / Internal | Identity lifecycle, dynamic JSON schema traits, Argon2id passwords, MFA. | [Ego Manual](/ego_usage_guide) |
| **Autorix Janus** | `4444` | HTTP REST / OIDC | Public / Internal | OAuth 2.0 PKCE, OpenID Connect tokens, automated RS256 JWKS rotation. | [Janus Manual](/janus_usage_guide) |
| **Autorix Vulcan**| `4466` | HTTP REST | Internal / Service | Prefixed API keys (`av_live_`), HMAC-SHA256 offline attenuated Macaroons. | [Vulcan Manual](/vulcan_usage_guide) |
| **Autorix Hermes**| `4477` | HTTP REST / XML | Public / IdP | Enterprise SAML 2.0 SP bridge and SCIM 2.0 directory sync (RFC 7644). | [Hermes Manual](/hermes_usage_guide) |
| **Prometheus** | `9090` | HTTP | Operations | Cluster-wide RED telemetry scraping across all 9 services. | [Operations Runbook](/operations_and_runbook) |

---

## 🔐 2. Credential & Token Lifecycle Standards

Autorix enforces recognizable credential prefixes across all components to simplify observability, secret scanning, and zero-trust verification:

| Credential Type | Prefix | Algorithm / Storage Format | Purpose & Security Scope |
| :--- | :--- | :--- | :--- |
| **Bootstrap Token** | `abt_` | High-entropy hex (Argon2id stored) | One-time initial claim of the Argus root master owner. |
| **Enrollment Token**| `aet_` | High-entropy hex (SHA-256 stored) | Authorizes new engine instances to join the cluster fleet. |
| **Session Token** | `ast_` | High-entropy hex (SHA-256 stored) | Operator session token for Argus and Console BFF. |
| **API Key (Live)** | `av_live_` | HMAC-SHA256 Chained Macaroon | Production machine-to-machine capability tokens with offline caveats. |
| **API Key (Test)** | `av_test_` | HMAC-SHA256 Chained Macaroon | Non-production integration testing capability tokens. |
| **Access Token** | `eyJ...` | RS256 Asymmetric RSA 2048-bit | OAuth 2.0 / OIDC JWT issued by Janus and verified via JWKS. |

---

## 🔄 3. End-to-End Multi-Engine Integration Workflows

### 3.1 Edge Request Pipeline (Aegis PEP ➔ Janus/Vulcan ➔ Nexus/Themis ➔ Upstream)

```text
[ Client ] ──(1. HTTP Request with Bearer Token or API Key)──► [ Aegis PEP :4455 ]
                                                                       │
             ┌─────────────────────────────────────────────────────────┤
             ▼ (2. Authenticate Token)                                 ▼ (3. Authorize Permission)
      [ Janus / Vulcan ]                                        [ Nexus / Themis ]
      - Validates JWT via JWKS                                  - Checks Zanzibar relation graph
      - Validates Macaroon Signature & Caveats                  - Evaluates CEL policy ring
             │                                                         │
             └─────────────────────────┬───────────────────────────────┘
                                       ▼ (4. Mutate Headers: X-User-ID, Claims)
                            [ Upstream Microservice ]
                                       │
                                       ▼ (5. Append Audit Event)
                              [ Argus Audit Chain ]
```

1. **Client** sends request to `https://api.enterprise.corp/resource` (handled by **Aegis PEP**).
2. **Aegis Authenticator** validates the caller identity using **Janus** (JWT JWKS public key) or **Vulcan** (Macaroon signature and first-party caveats).
3. **Aegis Authorizer** executes a sub-millisecond check against **Nexus** (ReBAC graph traversal) or **Themis** (ABAC CEL policy expression).
4. **Aegis Mutator** strips sensitive auth tokens, injects `X-User-ID`, `X-User-Email`, and custom context headers, then forwards the request to the upstream microservice.
5. The upstream service or Aegis records mutating actions to **Argus** (`POST /v1/audit`), cryptographically appending them to the SHA-256 Merkle chain.

---

### 3.2 Enterprise Identity & Directory Sync Flow (Hermes ➔ Ego ➔ Nexus)

```text
[ Enterprise IdP (Okta/Azure) ] ──(SCIM 2.0 Webhook)──► [ Hermes :4477 ]
                                                               │
                                                               ▼
                                                        [ Ego :4433 ]
                                            (Upserts Identity & Dynamic Traits)
                                                               │
                                                               ▼
                                                       [ Nexus :8080 ]
                                            (Writes group membership tuples)
```

1. When an employee is created or updated in Okta / Azure AD, the enterprise IdP pushes a SCIM 2.0 webhook to **Hermes**.
2. **Hermes** processes the event and synchronizes user traits and group assignments directly into **Ego**.
3. Group memberships are automatically mapped to **Nexus** relation tuples (`group:engineering#member@user:<id>`), immediately updating access graphs across the entire mesh.

---

## 📡 4. Standard Error Envelopes & Pagination

All Autorix REST APIs follow consistent error and pagination formats.

### Standard Error JSON Envelope
```json
{
  "error": "invalid_request",
  "error_description": "The 'namespace' parameter is required and cannot be empty."
}
```

### Standard Cursor Pagination Envelope
```json
{
  "data": [ ... ],
  "next_cursor": "eyJ0IjoiMjAyNi0wOC0yMFQwODozMDowMFoiLCJpZCI6IjBhMWI...\"}",
  "has_more": true
}
```
* **Page Request Query Parameters**: `?limit=50&cursor=<opaque_token>`
* **Filter Query Parameters**: `?filter.<field_name>=<value>`
