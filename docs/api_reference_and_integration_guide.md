# Master API Reference & Integration Guide: Autorix IAM Suite

This document serves as the authoritative integration manual and API reference for the complete **Autorix Zero-Trust IAM Ecosystem**.

---

# Table of Contents

1. [Architecture & Port Matrix](#1-architecture--port-matrix)
2. [Authentication & Credential Standards](#2-authentication--credential-standards)
3. [Service API Reference](#3-service-api-reference)
   - [3.1 Autorix Argus (Fleet Control Plane & Compliance :4400 / :50053)](#31-autorix-argus-fleet-control-plane--compliance)
   - [3.2 Autorix Themis (ABAC CEL Policy Engine :4488 / :50052)](#32-autorix-themis-abac-cel-policy-engine)
   - [3.3 Autorix Nexus (Google Zanzibar ReBAC Engine :8080 / :50051)](#33-autorix-nexus-google-zanzibar-rebac-engine)
   - [3.4 Autorix Ego (Identity, Traits & Argon2id :4433)](#34-autorix-ego-identity-traits--argon2id)
   - [3.5 Autorix Janus (OAuth 2.0 & OIDC Server :4444)](#35-autorix-janus-oauth-20--oidc-server)
   - [3.6 Autorix Aegis (Zero-Trust PEP Reverse Proxy :4455 / :4456)](#36-autorix-aegis-zero-trust-pep-reverse-proxy)
   - [3.7 Autorix Vulcan (API Keys & Attenuated Macaroons :4466)](#37-autorix-vulcan-api-keys--attenuated-macaroons)
   - [3.8 Autorix Hermes (SAML 2.0 & SCIM 2.0 Federation :4477)](#38-autorix-hermes-saml-20--scim-20-federation)
   - [3.9 Autorix Console BFF (:3000)](#39-autorix-console-bff)
4. [End-to-End Integration Flows](#4-end-to-end-integration-flows)

---

# 1. Architecture & Port Matrix

| Service | Port | Protocol | Ingress Tier | Purpose |
| :--- | :--- | :--- | :--- | :--- |
| **Aegis PEP** | `4455` | HTTP/1.1 | **Public / Edge** | Policy Enforcement Point (PEP) Reverse Proxy. |
| **Aegis Admin** | `4456` | HTTP REST | Internal | Dynamic YAML/Postgres proxy rules management. |
| **Console** | `3000` | HTTP / Next.js | **Public / Admin** | Zero-Trust Admin Console and BFF layer. |
| **Argus REST** | `4400` | HTTP REST | Admin / Internal | Fleet orchestration, tokens, audit hash chain, compliance. |
| **Argus gRPC** | `50053` | gRPC (Proto) | Cluster Internal | High-throughput engine heartbeats and enrollment. |
| **Themis REST**| `4488` | HTTP REST | Internal / Console | ABAC Google CEL policy CRUD & evaluation testing. |
| **Themis gRPC**| `50052` | gRPC (Proto) | Cluster Internal | In-cluster sub-millisecond CEL expression evaluation. |
| **Nexus REST** | `8080` | HTTP REST | Internal / Console | Relation tuple management and Check API. |
| **Nexus gRPC** | `50051` | gRPC (Proto) | Cluster Internal | Zanzibar relation graph traversal engine. |
| **Ego** | `4433` | HTTP REST | Public / Internal | User lifecycle, JSON traits, Argon2id passwords, TOTP MFA. |
| **Janus** | `4444` | HTTP REST / OIDC | Public / Internal | OAuth 2.0 PKCE, OpenID Connect tokens, JWKS. |
| **Vulcan** | `4466` | HTTP REST | Internal / SDK | API Keys (`av_live_...`), HMAC-SHA256 Macaroons. |
| **Hermes** | `4477` | HTTP REST / XML | Public / IdP | SAML 2.0 ACS, SCIM 2.0 Directory Sync (RFC 7643/7644). |
| **Prometheus** | `9090` | HTTP | Operations | Cluster-wide RED metrics scraper. |
| **Postgres** | `5432` | PostgreSQL | Database Tier | 8 partitioned databases with schema isolation. |

---

# 2. Authentication & Credential Standards

| Credential Type | Prefix | Algorithm / Format | Usage |
| :--- | :--- | :--- | :--- |
| **Bootstrap Token** | `abt_` | High-Entropy Hex (Argon2id stored) | One-time initial claim of Argus root owner. |
| **Enrollment Token** | `aet_` | High-Entropy Hex (SHA-256 stored) | Authorizes new engine instances to join cluster. |
| **Session Token** | `ast_` | High-Entropy Hex | Operator session cookie in Console / Argus. |
| **API Key (Live)** | `av_live_` | HMAC-SHA256 Macaroon Token | Production machine-to-machine capabilities. |
| **API Key (Test)** | `av_test_` | HMAC-SHA256 Macaroon Token | Non-production integration testing. |
| **JWT Access Token**| `eyJ...` | RS256 Asymmetric Signature | User authentication emitted by Janus OIDC. |

---

# 3. Service API Reference

---

## 3.1 Autorix Argus (Fleet Control Plane & Compliance)

### `GET /v1/auth/status`
Checks if the root owner has been initialized.

```bash
curl -s http://localhost:4400/v1/auth/status
```
```json
{
  "bootstrapped": true,
  "operators_count": 1
}
```

### `POST /v1/auth/bootstrap`
Initializes the root owner using the server bootstrap token.

```bash
curl -X POST http://localhost:4400/v1/auth/bootstrap \
  -H "Content-Type: application/json" \
  -d '{
    "bootstrap_token": "abt_0123456789abcdef0123456789abcdef",
    "name": "Cluster Administrator",
    "email": "admin@autorix.local",
    "password": "SecretMasterKey#2026"
  }'
```

### `POST /v1/auth/login`
Authenticates an administrative operator. Rate-limited to 5 attempts before 15-minute lockout.

```bash
curl -X POST http://localhost:4400/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@autorix.local",
    "password": "SecretMasterKey#2026"
  }'
```

### `GET /v1/audit`
Queries paginated immutable audit records.

```bash
curl -s "http://localhost:4400/v1/audit?limit=20&outcome=success"
```

### `GET /v1/audit/verify`
Runs cryptographic SHA-256 Merkle chain verification across all audit records.

```bash
curl -s http://localhost:4400/v1/audit/verify
```
```json
{
  "verified": true,
  "chain_length": 142,
  "head_hash": "a4f8e91c7b3d2e0f8a9c1b2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f",
  "verified_at": "2026-08-20T08:30:00Z",
  "algorithm": "SHA-256"
}
```

### `GET /v1/compliance/evidence`
Retrieves live compliance evaluation for SOC 2 and ISO 27001.

```bash
curl -s http://localhost:4400/v1/compliance/evidence
```

---

## 3.2 Autorix Themis (ABAC CEL Policy Engine)

### `POST /v1/policies`
Creates a new ABAC policy with Google Common Expression Language.

```bash
curl -X POST http://localhost:4488/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "Name": "Finance Department Wire Lockdown",
    "TenantID": "default",
    "Description": "Enforces MFA and business hours for wire transfers",
    "Expression": "request.auth.claims.department == \"finance\" && request.auth.mfa == true",
    "Priority": 1,
    "Enabled": true,
    "Labels": { "security": "high-assurance" }
  }'
```

### `POST /v1/policies/evaluate`
Evaluates CEL policies dynamically against a context.

```bash
curl -X POST http://localhost:4488/v1/policies/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "TenantID": "default",
    "Context": {
      "request": {
        "auth": { "claims": { "department": "finance" }, "mfa": true }
      }
    }
  }'
```

---

## 3.3 Autorix Nexus (Google Zanzibar ReBAC Engine)

### `POST /admin/relation-tuples`
Creates a relation tuple in the Zanzibar graph.

```bash
curl -X POST http://localhost:8080/admin/relation-tuples \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "documents",
    "object": "roadmap_q3_2026",
    "relation": "editor",
    "subject_namespace": "user",
    "subject_id": "usr_9988"
  }'
```

### `POST /v1/check`
Executes a ReBAC graph Check.

```bash
curl -X POST http://localhost:8080/v1/check \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "documents",
    "object": "roadmap_q3_2026",
    "relation": "editor",
    "subject_namespace": "user",
    "subject_id": "usr_9988"
  }'
```
```json
{
  "allowed": true
}
```

---

## 3.4 Autorix Ego (Identity & Lifecycle Engine)

### `POST /identities`
Registers a new user identity with traits.

```bash
curl -X POST http://localhost:4433/identities \
  -H "Content-Type: application/json" \
  -d '{
    "schema_id": "default",
    "traits": {
      "email": "engineer@enterprise.io",
      "name": "Jane Doe",
      "department": "Platform Engineering"
    },
    "credentials": {
      "password": { "config": { "password": "SecureUserPassword#2026" } }
    }
  }'
```

### `GET /sessions/whoami`
Validates active session cookie or Bearer token.

```bash
curl -s http://localhost:4433/sessions/whoami \
  -H "Cookie: ego_session=..."
```

---

## 3.5 Autorix Janus (OAuth 2.0 & OIDC Server)

### `GET /.well-known/openid-configuration`
OIDC discovery document.

```bash
curl -s http://localhost:4444/.well-known/openid-configuration
```

### `GET /.well-known/jwks.json`
Public RS256 JWKS keys for verifying JWT access tokens.

```bash
curl -s http://localhost:4444/.well-known/jwks.json
```

### `POST /oauth2/token`
Issues OAuth 2.0 access and refresh tokens.

```bash
curl -X POST http://localhost:4444/oauth2/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=client_abc&client_secret=secret_xyz&scope=read+write"
```

---

## 3.6 Autorix Aegis (Zero-Trust Reverse PEP Proxy)

### `GET /v1/rules` (`:4456` Admin API)
Lists active proxy routing and authorization rules.

```bash
curl -s http://localhost:4456/v1/rules
```

### `POST /v1/rules`
Creates a PEP interception rule:

```bash
curl -X POST http://localhost:4456/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "id": "protect-billing-api",
    "match": {
      "url": "http://localhost:4455/api/v1/billing/*",
      "methods": ["GET", "POST"]
    },
    "authenticators": [
      { "handler": "jwt", "config": { "jwks_url": "http://janus:4444/.well-known/jwks.json" } }
    ],
    "authorizers": [
      { "handler": "nexus", "config": { "namespace": "service", "relation": "access" } }
    ],
    "mutators": [
      { "handler": "header", "config": { "headers": { "X-User-ID": "{{ .Subject }}" } } }
    ],
    "upstream": {
      "url": "http://billing-service:8080"
    }
  }'
```

---

## 3.7 Autorix Vulcan (API Keys & Attenuated Macaroons)

### `POST /v1/keys`
Issues a high-entropy Macaroon API key:

```bash
curl -X POST http://localhost:4466/v1/keys \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Data Ingestion Pipeline",
    "environment": "live",
    "scopes": ["ingest:write", "datasets:read"],
    "expires_in": 2592000
  }'
```
```json
{
  "id": "key_8899aabb",
  "token": "av_live_9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a10",
  "prefix": "av_live_",
  "scopes": ["ingest:write", "datasets:read"],
  "expires_at": "2026-09-19T08:30:00Z"
}
```

### `POST /v1/keys/attenuate`
Attenuates an existing key with offline caveats (IP restriction, scope reduction, expiration):

```bash
curl -X POST http://localhost:4466/v1/keys/attenuate \
  -H "Content-Type: application/json" \
  -d '{
    "token": "av_live_9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a10",
    "caveats": [
      "time < 2026-08-21T00:00:00Z",
      "ip = 10.0.4.15",
      "scope = ingest:write"
    ]
  }'
```

---

## 3.8 Autorix Hermes (SAML 2.0 & SCIM 2.0 Federation)

### `GET /v1/saml/metadata`
Returns Hermes Service Provider SAML 2.0 XML metadata for Okta/Entra ID import.

```bash
curl -s http://localhost:4477/v1/saml/metadata
```

### `GET /scim/v2/Users`
SCIM 2.0 user directory sync endpoint (RFC 7644).

```bash
curl -s http://localhost:4477/scim/v2/Users \
  -H "Authorization: Bearer <scim_bearer_token>"
```

---

## 3.9 Autorix Console BFF

| Endpoint | Method | Purpose |
| :--- | :--- | :--- |
| `/api/auth/status` | `GET` | Returns bootstrap status from Argus. |
| `/api/auth/login` | `POST` | Authenticates operator, sets HTTP-only session cookie. |
| `/api/auth/logout` | `POST` | Invalidates session cookie and server session. |
| `/api/notifications` | `GET` | Aggregates live fleet audit and instance alerts for Notification Center. |
| `/api/audit` | `GET`, `POST` | Proxies audit records to Argus with double-submit CSRF. |
| `/api/audit/verify` | `GET` | Requests SHA-256 Merkle chain verification from Argus. |
| `/api/compliance/evidence` | `GET` | Proxies live compliance controls evaluation. |
| `/api/fleet/instances` | `GET` | Lists live fleet instances. |
| `/api/health` | `GET` | Measures status and latency across all 7 engines + Argus. |
