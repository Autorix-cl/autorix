# Autorix Vulcan: API Keys & Macaroon Attenuation Manual

**Autorix Vulcan** is an enterprise API Key and decentralized capability token engine inspired by Google Macaroons and Ory Talos. It issues high-entropy, environment-prefixed API keys that support **offline, client-side caveat attenuation** using HMAC-SHA256 cryptographic chaining.

---

## 🏛️ 1. Core Architecture & Macaroon Cryptography

```text
  [ Master Client / SDK ] ──(Attenuates token offline)──► [ Distributed Worker / Lambda ]
                                                                 │
                                                                 ▼ (Calls API)
  ┌────────────────────────────────────────────────────────────────────────┐
  │                            Autorix Vulcan                              │
  │                                                                        │
  │  ┌──────────────────────┐  ┌────────────────────────────────────────┐  │
  │  │ Prefixed Key Minter  │  │ HMAC-SHA256 Chained Macaroon Verifier  │  │
  │  │ (av_live_ / av_test_)│  │ (Evaluates caveats & verifies sig)     │  │
  │  └──────────┬───────────┘  └───────────────────┬────────────────────┘  │
  └─────────────┼──────────────────────────────────┼───────────────────────┘
                │                                  │
                ▼                                  ▼
            [ PostgreSQL Database: api_keys, scopes, usage_metrics ]
```

### 1.1 Why Macaroons instead of Plain API Keys?

* **Decentralized Attenuation**: A root API key issued to a primary backend service can be restricted ("attenuated") by that service before passing it to a worker thread, lambda function, or untrusted third party, **without making any database calls to Vulcan**.
* **Cryptographic Tamper-Proofing**: Each caveat appended to a Macaroon updates the HMAC signature:
  ```text
  Sig_0   = HMAC-SHA256(K_root, KeyID)
  Sig_i+1 = HMAC-SHA256(Sig_i, Caveat_i+1)
  ```
  An attacker cannot remove or alter a caveat without invalidating the entire cryptographic signature chain.

---

## 🔑 2. Environment Prefixes & Key Storage

Vulcan enforces standard prefixes to enable secrets scanners (GitHub Secret Scanning, TruffleHog) to detect leaked credentials:

| Prefix | Environment | Purpose |
| :--- | :--- | :--- |
| `av_live_` | Production | Live production access with real customer data. |
| `av_test_` | Staging / Sandbox | Mock data and non-production testing. |

### Storage Security at Rest

* Vulcan **never stores plaintext keys** in PostgreSQL.
* The database stores `key_hash = SHA-256(raw_key)` and `key_hint = raw_key[len-4:]`.
* Key verification uses constant-time comparison (`crypto/subtle.ConstantTimeCompare`) to eliminate timing attacks.

---

## 📡 3. Complete REST API Reference

Vulcan runs on port `4466`.

### 3.1 Create API Key (`POST /keys`)

* **Method**: `POST`
* **Path**: `/keys`
* **Headers**: `Content-Type: application/json`

#### Request Body
```json
{
  "name": "ETL Ingestion Pipeline",
  "description": "Daily metrics aggregator",
  "owner_id": "svc_etl_runner",
  "scopes": ["ingest:write", "metrics:read"],
  "is_live": true,
  "expires_at": "2026-12-31T23:59:59Z"
}
```

#### Response (`201 Created`)
```json
{
  "api_key": {
    "id": "c1a2b3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
    "key_prefix": "av_live",
    "key_hint": "9f8e",
    "name": "ETL Ingestion Pipeline",
    "owner_id": "svc_etl_runner",
    "scopes": ["ingest:write", "metrics:read"],
    "state": "active",
    "created_at": "2026-08-20T08:00:00Z"
  },
  "raw_token": "av_live_9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a10e9f8a7b6c5d4e3f2a1b0c9",
  "macaroon": {
    "location": "http://localhost:4466",
    "key_id": "c1a2b3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
    "caveats": [],
    "signature": "a4f8e91c7b3d2e0f8a9c1b2d3e4f5a6b..."
  }
}
```

---

### 3.2 List API Keys (`GET /keys`)

Lists metadata of all issued API keys (never exposing raw secrets or root signature keys) with cursor pagination.

---

### 3.3 Attenuate Macaroon (`POST /keys/attenuate`)

Attenuates a Macaroon by appending a first-party caveat and returning the updated token.

* **Method**: `POST`
* **Path**: `/keys/attenuate`

#### Request Body
```json
{
  "macaroon": {
    "location": "http://localhost:4466",
    "key_id": "c1a2b3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
    "caveats": [],
    "signature": "a4f8e91c7b3d2e0f8a9c1b2d3e4f5a6b..."
  },
  "caveat": "time_before = 2026-08-21T00:00:00Z"
}
```

---

### 3.4 Verify Macaroon (`POST /keys/verify`)

Evaluates the cryptographic signature chain and tests all caveat assertions against the provided runtime context.

* **Method**: `POST`
* **Path**: `/keys/verify`

#### Request Body
```json
{
  "macaroon": {
    "location": "http://localhost:4466",
    "key_id": "c1a2b3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
    "caveats": [
      { "clause": "time_before = 2026-08-21T00:00:00Z" }
    ],
    "signature": "b5e9f02d8c4e3f1a..."
  },
  "context": {
    "now": "2026-08-20T14:30:00Z",
    "ip_address": "10.0.4.15"
  }
}
```

#### Response (`200 OK`)
```json
{
  "valid": true,
  "api_key": {
    "id": "c1a2b3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
    "name": "ETL Ingestion Pipeline",
    "owner_id": "svc_etl_runner",
    "scopes": ["ingest:write", "metrics:read"]
  }
}
```

---

### 3.5 Admin: Rotate Key (`POST /admin/keys/{id}/rotate`)

Generates a new root signature key while preserving the previous key during a configurable grace period:

```bash
curl -X POST http://localhost:4466/admin/keys/c1a2b3d4-e5f6.../rotate \
  -H "Content-Type: application/json" \
  -d '{ "grace_period": "48h" }'
```

---

### 3.6 Admin: Revoke Key & Scopes Catalogue

- `DELETE /keys/{id}`: Immediately revokes an API key across the cluster.
- `GET /admin/scopes`: Lists system scopes.
- `POST /admin/scopes`: Creates a scope.
- `DELETE /admin/scopes/{name}`: Deletes a scope.

---

## 🛠️ 4. Production Recipes

### Offline Worker Delegation Recipe
1. Primary service receives root Macaroon `M0`.
2. Primary service attenuates `M0` with `time_before = NOW + 10m` and `method = POST` creating `M1` locally in memory using the Autorix SDK.
3. Primary service passes `M1` to worker thread.
4. Worker calls API using `M1`. Vulcan verifies `M1` without any database lookup to fetch extra permissions.
