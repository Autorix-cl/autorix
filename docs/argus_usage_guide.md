# Autorix Argus: Fleet Control Plane, Audit Trail & Compliance Guide

**Autorix Argus** is the central governance and fleet orchestration control plane of the Autorix Zero-Trust IAM suite. It coordinates engine registration, cryptographic enrollment tokens (`aet_...`), cluster topology, administrative sessions (`ast_...`), tamper-evident SHA-256 chained audit logs, and continuous SOC 2 / ISO 27001 compliance evidence.

---

## 🏗️ 1. Architecture & Core Responsibilities

```text
  ┌─────────────────────────────────────────────────────────────┐
  │                        Autorix Argus                        │
  │                                                             │
  │  ┌───────────────────┐  ┌───────────────────┐  ┌─────────┐  │
  │  │  Fleet Management │  │  Identity & Auth  │  │  Audit  │  │
  │  │  & Enrollment     │  │  & Sessions       │  │  Chain  │  │
  │  │  (aet_... Tokens) │  │  (Argon2id/Tokens)│  │ (SHA256)│  │
  │  └─────────┬─────────┘  └─────────┬─────────┘  └────┬────┘  │
  │            │                      │                 │       │
  │  ┌─────────▼──────────────────────▼─────────────────▼────┐  │
  │  │          Continuous Compliance Evidence Engine        │  │
  │  │          (SOC 2 CC6.1/CC6.2 & ISO 27001)              │  │
  │  └────────────────────────┬──────────────────────────────┘  │
  └───────────────────────────┼─────────────────────────────────┘
                              │
                              ▼
                      [ PostgreSQL DB ]
                      (autorix_argus)
```

1. **Fleet Orchestration & Heartbeats**: Tracks active instances of all 7 Autorix engines, monitoring latency, build SHAs, and dependency topologies via gRPC streaming (`:50053`).
2. **Cryptographic Enrollment Tokens**: Issues high-entropy `aet_...` tokens scoped to engine types and environments for zero-trust provisioning.
3. **Control Plane Operators & Sessions**: Authenticates administrative operators with Argon2id passwords, rate-limiting failed logins (5 attempts $\rightarrow$ 15-minute lock), and issues `ast_...` session tokens.
4. **Tamper-Evident SHA-256 Chained Audit Trail**: Persists every administrative action in an append-only table where every entry contains the SHA-256 hash of the preceding entry.
5. **Continuous Compliance Evidence**: Automatically computes compliance metrics and verifiable proofs for SOC 2 Type II and ISO 27001 controls.

---

## 🔑 2. Enrollment Token Lifecycle

Engines authenticate with Argus using high-entropy Enrollment Tokens (`aet_` prefix).

### 2.1 Minting an Enrollment Token (`POST /v1/enrollment-tokens`)

```bash
curl -X POST http://localhost:4400/v1/enrollment-tokens \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <session_token>" \
  -d '{
    "engine_type": "nexus",
    "environment": "production",
    "uses_allowed": 1,
    "expires_in_seconds": 86400,
    "created_by": "admin@autorix.local"
  }'
```

**Response (`201 Created`):**
```json
{
  "id": "e85e0794-060b-41bf-a428-2d75c8cccd0e",
  "token": "aet_01917f8a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0",
  "engine_type": "nexus",
  "environment": "production",
  "expires_at": "2026-08-21T08:00:00Z",
  "uses_allowed": 1,
  "uses_count": 0
}
```

---

## 🛡️ 3. Immutable Cryptographic Audit Trail

Every mutating action creates a linked record computed across:
```text
Record Hash = SHA-256(prev_hash + id + action + resource_type + resource_id + outcome + created_at)
```

### 3.1 Verifying Audit Chain Integrity (`GET /v1/audit/verify`)

```bash
curl -s http://localhost:4400/v1/audit/verify
```

**Response (`200 OK`):**
```json
{
  "verified": true,
  "chain_length": 142,
  "head_hash": "a4f8e91c7b3d2e0f8a9c1b2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f",
  "algorithm": "SHA-256",
  "verified_at": "2026-08-20T14:30:00Z"
}
```

---

## 📡 4. Complete REST & gRPC API Reference

Argus exposes REST on port `4400` and gRPC on port `50053`.

### 4.1 Fleet & Instance Management

- `GET /v1/instances`: Lists registered engine instances with filtering (`?engine_type=`, `?status=`, `?environment=`) and cursor pagination.
- `GET /v1/instances/{id}`: Returns instance details, recent events, and dependency graph.
- `POST /v1/instances/manual`: Registers an unverified manual instance when automated tokens cannot be used.
- `POST /v1/instances/{id}/force-remove`: Administrative immediate eviction and credential revocation.
- `GET /v1/topology`: Returns the full cluster dependency graph.
- `GET /v1/stream`: Real-time Server-Sent Events (SSE) stream of instance state updates.

---

### 4.2 Enrollment Tokens API

- `POST /v1/enrollment-tokens`: Mints a new `aet_` token.
- `GET /v1/enrollment-tokens`: Lists active enrollment tokens.
- `DELETE /v1/enrollment-tokens/{id}`: Revokes an enrollment token.
- `GET /v1/enrollment-audit`: Retrieves audit history of token minting, consumption, and revocations.

---

### 4.3 Operator Auth & Sessions

- `GET /v1/auth/status`: Checks if initial root administrator has been bootstrapped.
- `POST /v1/auth/bootstrap`: Claims the cluster using the one-time `abt_` token.
- `POST /v1/auth/login`: Authenticates operator and issues an `ast_` session token.
- `GET /v1/auth/session`: Validates and touches an active operator session.
- `DELETE /v1/auth/session`: Logs out operator and revokes session.
- `GET /v1/operators`: Lists administrative operators.

---

### 4.4 Audit & Compliance Evidence

- `GET /v1/audit`: Paginated query across audit records (`?actor_id=`, `?action=`, `?resource_type=`, `?outcome=`).
- `POST /v1/audit`: Appends an external audit event to the hash chain.
- `GET /v1/audit/verify`: Executes mathematical SHA-256 chain verification.
- `GET /v1/audit/export`: Exports audit log formatted as JSON or CSV (`?format=csv`).
- `GET /v1/compliance/evidence`: Generates SOC 2 CC6.1/CC6.2 and ISO 27001 evidence report.
- `GET /v1/environments`: Lists cluster environments.
- `GET /v1/governance/orgs`: Lists governance organizations.
- `GET /v1/governance/projects`: Lists governance projects.

---

### 4.5 gRPC Control Interface (`:50053`)

Argus implements `argus.v1.ArgusControlService` for cluster engines:
- `Register(RegistrationRequest) returns (RegistrationResponse)`
- `Heartbeat(HeartbeatRequest) returns (HeartbeatResponse)`
- `ListInstances(ListInstancesRequest) returns (ListInstancesResponse)`
- `GetInstance(GetInstanceRequest) returns (GetInstanceResponse)`
- `Evict(EvictRequest) returns (EvictResponse)`

---

## 🛠️ 5. Production Recipes

### Initial Cluster Claim Recipe
1. Fetch the bootstrap token from Docker logs:
   ```bash
   docker logs autorix-argus | grep "Bootstrap token generated"
   ```
2. Initialize root administrator:
   ```bash
   curl -X POST http://localhost:4400/v1/auth/bootstrap \
     -H "Content-Type: application/json" \
     -d '{
       "bootstrap_token": "abt_0123456789abcdef...",
       "name": "Cluster Administrator",
       "email": "admin@autorix.local",
       "password": "SecretMasterKey#2026"
     }'
   ```
