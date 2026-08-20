# Autorix Argus: Control Plane, Audit Trail & Compliance Guide

**Autorix Argus** is the central management and fleet orchestration engine of the Autorix Zero-Trust IAM suite. It coordinates engine registration, enrollment tokens, cluster topology, operator credentials and sessions, tamper-evident SHA-256 chained audit trails, and continuous compliance evaluation.

---

## 🏗️ Architecture & Core Responsibilities

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

1. **Engine Fleet Registry**: Manages instances of all 7 Autorix engines (Nexus, Ego, Janus, Aegis, Vulcan, Hermes, Themis), tracking heartbeat status, latency, build SHAs, endpoints, and dependency graphs.
2. **Cryptographic Enrollment Tokens**: Issues high-entropy `aet_...` tokens scoped to engine types and environments for zero-trust engine provisioning.
3. **Control Plane Operators & Sessions**: Authenticates administrative operators with Argon2id passwords, implements rate limiting and automatic lockouts (5 failed attempts $\rightarrow$ 15 min lock), and issues high-entropy session tokens (`ast_...`).
4. **Immutable Audit Trail with Hash Chaining**: Persists every administrative action in an append-only table (`audit_records`) where each record contains the SHA-256 hash of the previous record, enabling mathematical verification against tampering.
5. **Continuous Compliance Evidence Aggregator**: Gathers real-time evidence for SOC 2 Type II and ISO 27001 audits (audit chain integrity, operator counts, active instances).

---

## 🔑 Enrollment Token Lifecycle

Engines authenticate with Argus using single-use or multi-use Enrollment Tokens with the `aet_` prefix.

### 1. Minting an Enrollment Token

```bash
curl -X POST http://localhost:4400/v1/enrollment-tokens \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <operator_token>" \
  -d '{
    "engine_type": "nexus",
    "environment": "production",
    "expires_in_seconds": 86400,
    "max_uses": 1
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
  "max_uses": 1,
  "uses_count": 0
}
```

### 2. Passing Token to Engine

Engines receive the token via environment variable:
```bash
AUTORIX_ENROLLMENT_TOKEN="aet_01917f8a..." ./nexusd
```

### 3. Verification & Audit Recording

When the engine starts, it calls Argus's gRPC `EnrollInstance` endpoint (`:50053`). Argus validates the token hash, records an audit event in `enrollment_audit_log` (`mint`, `consume`, `consume_failed`, `revoke`), and registers the instance.

---

## 🛡️ Immutable Cryptographic Audit Trail

Argus implements an append-only Merkle-style hash chain (P8-S1 standard). Every record contains:
* `id`: Unique UUID
* `actor_id` & `actor_type`: Operator email or service name
* `action`: Operation performed (`CREATE`, `UPDATE`, `DELETE`, `LOGIN`, `MINT`, `REVOKE`)
* `resource_type` & `resource_id`: Target entity
* `before_state` & `after_state`: JSON diff of changes (redacted of secrets)
* `prev_hash`: SHA-256 hash of the previous record
* `record_hash`: SHA-256 hash computed across:
  $$\text{SHA-256}(\text{prev\_hash} + \text{id} + \text{action} + \text{resource\_type} + \text{resource\_id} + \text{outcome} + \text{created\_at})$$

### Verifying Audit Trail Integrity

To verify that no database record has been altered, deleted, or inserted out of order:

```bash
curl -s http://localhost:4400/v1/audit/verify
```

**Response:**
```json
{
  "verified": true,
  "chain_length": 142,
  "head_hash": "a4f8e91c7b3d2e0f8a9c1b2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f",
  "genesis_hash": "0000000000000000000000000000000000000000000000000000000000000000",
  "verified_at": "2026-08-20T08:30:00Z",
  "algorithm": "SHA-256"
}
```

If a record is modified by an attacker in PostgreSQL directly, `verified` returns `false` with the exact `broken_link` index.

### Exporting Audit Records

```bash
# JSON Export
curl -s http://localhost:4400/v1/audit/export?format=json > audit-records.json

# CSV Export for Compliance Auditors
curl -s http://localhost:4400/v1/audit/export?format=csv > audit-records.csv
```

---

## 📊 Continuous Compliance Evidence API

Argus continuously evaluates compliance controls and exposes evidence reports:

```bash
curl -s http://localhost:4400/v1/compliance/evidence
```

**Response:**
```json
{
  "data": [
    {
      "id": "CTRL-AUD-001",
      "framework": "SOC2",
      "control_id": "CC6.1",
      "control_name": "Cryptographic Audit Trail Integrity",
      "status": "compliant",
      "evidence_type": "hash_chain_verification",
      "description": "All security events and administrative actions are hashed and chained using SHA-256.",
      "engine": "argus",
      "last_evaluated_at": "2026-08-20T08:30:00Z",
      "evaluator": "argus-compliance-engine",
      "artifacts_count": 142,
      "details": {
        "algorithm": "SHA-256",
        "chain_length": 142,
        "chain_verified": true,
        "head_hash": "a4f8e91c..."
      }
    },
    {
      "id": "CTRL-IAM-001",
      "framework": "SOC2",
      "control_id": "CC6.2",
      "control_name": "Administrative Operator Access Review",
      "status": "compliant",
      "evidence_type": "operator_inventory",
      "description": "Operator accounts and privileged credentials enrolled in control plane.",
      "engine": "argus",
      "last_evaluated_at": "2026-08-20T08:30:00Z",
      "evaluator": "argus-compliance-engine",
      "artifacts_count": 1,
      "details": {
        "active_operators": 1
      }
    }
  ],
  "summary": {
    "total_controls": 3,
    "compliant_controls": 3,
    "review_required": 0,
    "score_percent": 100
  }
}
```

---

## 📡 REST API Reference

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/v1/auth/status` | Returns whether root owner has been initialized (`bootstrapped`). |
| `POST` | `/v1/auth/bootstrap` | Claims cluster using `abt_...` bootstrap token to create root owner. |
| `POST` | `/v1/auth/login` | Authenticates operator, checks lockouts, returns session token. |
| `GET` | `/v1/auth/session` | Validates session token and touches idle expiry. |
| `DELETE` | `/v1/auth/session` | Revokes active operator session. |
| `GET` | `/v1/operators` | Lists registered operators and their roles. |
| `GET` | `/v1/instances` | Lists registered fleet instances and health states. |
| `GET` | `/v1/instances/{id}` | Detailed instance status and dependency edges. |
| `POST` | `/v1/enrollment-tokens` | Mints a new engine enrollment token. |
| `GET` | `/v1/enrollment-tokens` | Lists active enrollment tokens. |
| `DELETE` | `/v1/enrollment-tokens/{id}` | Revokes an enrollment token. |
| `GET` | `/v1/enrollment-audit` | Lists audit log of token mints and enrollments. |
| `GET` | `/v1/topology` | Returns cluster graph with engine dependency relationships. |
| `GET` | `/v1/stream` | Server-Sent Events (SSE) stream of live fleet status transitions. |
| `GET` | `/v1/audit` | Paginated query of immutable audit log records. |
| `POST` | `/v1/audit` | Records a new chained audit event. |
| `GET` | `/v1/audit/verify` | Runs full SHA-256 hash chain verification. |
| `GET` | `/v1/audit/export` | Exports audit records in JSON or CSV format. |
| `GET` | `/v1/compliance/evidence` | Returns continuous SOC 2/ISO compliance assessment. |
| `GET` | `/v1/environments` | Lists configured environments (Production, Staging, etc.). |
| `GET` | `/v1/governance/orgs` | Lists tenant organizations in multi-tenant mode. |
| `GET` | `/v1/governance/projects` | Lists projects partitioned under organizations. |
