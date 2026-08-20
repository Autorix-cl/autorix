# Autorix Vulcan: API Keys & Macaroon Attenuation Manual

**Autorix Vulcan** is a high-assurance API Key and capability token management engine inspired by Ory Talos and Google Macaroons. It issues environment-prefixed, high-entropy API keys that support **offline, client-side caveat attenuation** using HMAC-SHA256 cryptographic chaining.

---

## 🏛️ 1. Core Architecture & Macaroon Cryptography

```text
    [ Client / SDK ] ──(Issues attenuated key offline)──► [ Distributed Worker ]
                                                                   │
                                                                   ▼ (Calls API)
    ┌────────────────────────────────────────────────────────────────────────┐
    │                            Autorix Vulcan                              │
    │                                                                        │
    │  ┌──────────────────────┐  ┌────────────────────────────────────────┐  │
    │  │ Prefixed Key Minter  │  │ HMAC-SHA256 Chained Macaroon Verifier  │  │
    │  │ (av_live_ / av_test_)│  │ (Evaluates first-party caveats)        │  │
    │  └──────────┬───────────┘  └───────────────────┬────────────────────┘  │
    └─────────────┼──────────────────────────────────┼───────────────────────┘
                  │                                  │
                  ▼                                  ▼
              [ PostgreSQL: api_keys, scopes, revocations ]
```

### 1.1 Why Macaroons instead of Plain API Keys?

* **Decentralized Attenuation**: A root API key issued to a primary backend service can be restricted ("attenuated") by that service before passing it to a worker thread, lambda function, or untrusted third-party, **without making any database calls to Vulcan**.
* **Cryptographic Tamper-Proofing**: Each caveat appended to a Macaroon updates the HMAC signature:
  $$\text{Sig}_0 = \text{HMAC-SHA256}(K_{\text{secret}}, \text{KeyID})$$
  $$\text{Sig}_{i+1} = \text{HMAC-SHA256}(\text{Sig}_i, \text{Caveat}_{i+1})$$
  An attacker cannot remove a caveat without invalidating the entire cryptographic signature chain.

---

## 🔑 2. Environment Prefixes & Key Storage

Vulcan enforces standard prefixes to enable secrets scanners (e.g. GitHub Secret Scanning, TruffleHog) to detect leaked credentials:

| Prefix | Environment | Purpose |
| :--- | :--- | :--- |
| `av_live_` | Production | Live production access with real data. |
| `av_test_` | Staging / Sandbox | Mock data and non-production testing. |

### Storage Security at Rest

* Vulcan **never stores plaintext keys** in PostgreSQL.
* The database stores `token_hash = SHA-256(raw_key)` and `key_prefix = raw_key[0:12]`.
* Key verification is constant-time (`crypto/subtle.ConstantTimeCompare`) to prevent timing attacks.

---

## 🚀 3. Complete API Reference

### 3.1 Issuing a New Key (`POST /keys`)

```bash
curl -X POST http://localhost:4466/keys \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ETL Ingestion Pipeline",
    "environment": "live",
    "scopes": ["ingest:write", "datasets:read", "metrics:write"],
    "expires_in_seconds": 2592000
  }'
```

**Response (`201 Created`):**
```json
{
  "id": "key_01918a7b-6c5d-4e3f-2a1b-0c9d8e7f6a50",
  "token": "av_live_9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a10e9f8a7b6c5d4e3f2a1b0c9",
  "name": "ETL Ingestion Pipeline",
  "key_prefix": "av_live_9f8e",
  "scopes": ["ingest:write", "datasets:read", "metrics:write"],
  "expires_at": "2026-09-19T09:00:00Z",
  "created_at": "2026-08-20T09:00:00Z"
}
```

### 3.2 Attenuating a Key with Caveats (`POST /keys/attenuate` or Client SDK)

Clients can restrict capabilities by adding caveats:

```bash
curl -X POST http://localhost:4466/keys/attenuate \
  -H "Content-Type: application/json" \
  -d '{
    "token": "av_live_9f8e7d6c5b4a3f2e...",
    "caveats": [
      "time < 2026-08-21T00:00:00Z",
      "ip = 10.0.4.15",
      "method = POST",
      "path = /api/v1/ingest/metrics",
      "scope = metrics:write"
    ]
  }'
```

**Response (`200 OK`):**
```json
{
  "attenuated_token": "av_live_9f8e7d6c5b4a3f2e...[chained signature]...",
  "caveats_applied": [
    "time < 2026-08-21T00:00:00Z",
    "ip = 10.0.4.15",
    "method = POST",
    "path = /api/v1/ingest/metrics",
    "scope = metrics:write"
  ]
}
```

### 3.3 Verifying an API Key (`POST /keys/verify`)

Called by Aegis or internal microservices to validate incoming keys:

```bash
curl -X POST http://localhost:4466/keys/verify \
  -H "Content-Type: application/json" \
  -d '{
    "token": "av_live_9f8e7d6c5b4a3f2e...",
    "context": {
      "ip": "10.0.4.15",
      "method": "POST",
      "path": "/api/v1/ingest/metrics",
      "required_scope": "metrics:write"
    }
  }'
```

**Response (`200 OK`):**
```json
{
  "valid": true,
  "key_id": "key_01918a7b-6c5d-4e3f-2a1b-0c9d8e7f6a50",
  "name": "ETL Ingestion Pipeline",
  "scopes": ["metrics:write"],
  "environment": "live"
}
```

### 3.4 Key Rotation & Revocation

* `POST /admin/keys/{id}/rotate`: Rotates a key secret immediately while keeping metadata and scopes intact.
* `DELETE /keys/{id}`: Permanently revokes the key and all attenuated child tokens derived from it.
* `GET /admin/scopes`: Lists available scopes in the system.
