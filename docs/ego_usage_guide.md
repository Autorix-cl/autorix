# Autorix Ego: Identity, Traits & Session Management Manual

**Autorix Ego** is an enterprise identity and user lifecycle engine inspired by Ory Kratos. It provides dynamic JSON Schema-driven identity traits, memory-hard Argon2id password hashing, TOTP multi-factor authentication (MFA), recovery codes, and high-performance session validation.

---

## 🏛️ 1. Architecture & Core Concepts

```text
  ┌─────────────────────────────────────────────────────────────┐
  │                         Autorix Ego                         │
  │                                                             │
  │  ┌────────────────────┐  ┌───────────────────────────────┐  │
  │  │ Dynamic Traits     │  │ Argon2id Password Hasher      │  │
  │  │ JSON Schema Engine │  │ (m=64MB, t=3, p=4)            │  │
  │  └─────────┬──────────┘  └───────────────┬───────────────┘  │
  │            │                             │                  │
  │  ┌─────────▼──────────┐  ┌───────────────▼───────────────┐  │
  │  │ Session Manager    │  │ TOTP MFA (RFC 6238)           │  │
  │  │ (Idle & Max TTL)   │  │ & Recovery Codes Vault        │  │
  │  └─────────┬──────────┘  └───────────────┬───────────────┘  │
  └────────────┼─────────────────────────────┼──────────────────┘
               │                             │
               ▼                             ▼
                    [ PostgreSQL Database ]
                        (autorix_ego)
```

### 1.1 Identity Schema Architecture

Ego does not hardcode user columns in SQL. Instead, user attributes are modeled as **Traits** validated against **JSON Schema (Draft 7)**:
* Standard fields: `email`, `name`, `username`.
* Custom enterprise fields: `department`, `employee_id`, `cost_center`, `security_clearance`, `billing_address`.

---

## 🔐 2. Cryptographic Security Standards

### 2.1 Argon2id Password Hashing

Ego strictly uses Argon2id (the winner of the Password Hashing Competition) to defend against GPU/ASIC brute-force cracking:

| Parameter | Value | Rationale |
| :--- | :--- | :--- |
| **Memory (`m`)** | `65536 KiB` (64 MB) | Forces high memory consumption per thread, rendering GPU cluster cracking infeasible. |
| **Iterations (`t`)** | `3` | Optimal balance between CPU resistance and user login latency (~150ms). |
| **Parallelism (`p`)** | `4 threads` | Utilizes multi-core architecture. |
| **Salt Length** | `16 bytes` | CSPRNG cryptographically secure random salt. |
| **Key Length** | `32 bytes` | Output hash size. |

### 2.2 TOTP MFA (RFC 6238)

* **Algorithm**: HMAC-SHA1 with 30-second time steps and 6-digit codes.
* **Secret Storage**: Base32 encoded, encrypted at rest.
* **Recovery Codes**: 8 single-use cryptographically random backup codes generated upon MFA activation.

---

## 📜 3. Dynamic JSON Schema Configuration

Schemas are stored in `identity_schemas` and applied during user registration, update, and import.

### 3.1 Creating an Identity Schema

```bash
curl -X POST http://localhost:4433/admin/schemas \
  -H "Content-Type: application/json" \
  -d '{
    "id": "enterprise_employee_v1",
    "name": "Enterprise Employee Profile",
    "schema": {
      "$schema": "http://json-schema.org/draft-07/schema#",
      "type": "object",
      "properties": {
        "email": {
          "type": "string",
          "format": "email"
        },
        "name": {
          "type": "string",
          "minLength": 2
        },
        "department": {
          "type": "string",
          "enum": ["Engineering", "Product", "Finance", "Security", "Legal"]
        },
        "employee_id": {
          "type": "string",
          "pattern": "^EMP-[0-9]{4}$"
        }
      },
      "required": ["email", "name", "department"],
      "additionalProperties": false
    }
  }'
```

---

## 🚀 4. Complete API Reference

### 4.1 Self-Service Registration (`POST /self-service/registration`)

```bash
curl -X POST http://localhost:4433/self-service/registration \
  -H "Content-Type: application/json" \
  -d '{
    "schema_id": "enterprise_employee_v1",
    "traits": {
      "email": "sarah.connor@cyberdyne.com",
      "name": "Sarah Connor",
      "department": "Security",
      "employee_id": "EMP-1001"
    },
    "password": "UltraSecretPassword#2026"
  }'
```

**Response (`201 Created`):**
```json
{
  "identity": {
    "id": "7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",
    "schema_id": "enterprise_employee_v1",
    "state": "active",
    "traits": {
      "email": "sarah.connor@cyberdyne.com",
      "name": "Sarah Connor",
      "department": "Security",
      "employee_id": "EMP-1001"
    },
    "created_at": "2026-08-20T09:00:00Z"
  }
}
```

### 4.2 Self-Service Login (`POST /self-service/login`)

```bash
curl -i -X POST http://localhost:4433/self-service/login \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "sarah.connor@cyberdyne.com",
    "password": "UltraSecretPassword#2026"
  }'
```

**Response (`200 OK` + Session Cookie):**
```text
HTTP/1.1 200 OK
Set-Cookie: autorix_session_token=ast_01918a7b6c5d4e3f...; Path=/; HttpOnly; SameSite=Lax
Content-Type: application/json

{
  "session_token": "ast_01918a7b6c5d4e3f...",
  "identity": {
    "id": "7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",
    "traits": { "email": "sarah.connor@cyberdyne.com" }
  },
  "expires_at": "2026-08-27T09:00:00Z"
}
```

### 4.3 Validate Session (`GET /sessions/whoami`)

Called by API Gateways and downstream services to authenticate the caller:

```bash
curl -s http://localhost:4433/sessions/whoami \
  -H "Authorization: Bearer ast_01918a7b6c5d4e3f..."
```

**Response:**
```json
{
  "id": "sess_998877",
  "active": true,
  "identity": {
    "id": "7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",
    "traits": {
      "email": "sarah.connor@cyberdyne.com",
      "name": "Sarah Connor",
      "department": "Security"
    }
  },
  "authenticated_at": "2026-08-20T09:00:00Z",
  "expires_at": "2026-08-27T09:00:00Z"
}
```

### 4.4 Admin Identity Management

* `GET /admin/identities?q=sarah&state=active&limit=25`
* `GET /admin/identities/{id}`
* `PATCH /admin/identities/{id}`: Update traits or identity state (`active`, `suspended`, `locked`).
* `DELETE /admin/identities/{id}`: Soft or hard delete.
* `POST /admin/identities/{id}/credentials/reset-password`: Sets a new password directly.
* `POST /admin/identities/{id}/recovery-link`: Mints an expiring one-time password reset token.

### 4.5 Admin Session Revocation

* `DELETE /admin/sessions/{id}`: Revokes a single specific session.
* `DELETE /admin/identities/{id}/sessions`: Revokes all active sessions for a user (useful for security incident response).

---

## ⏱️ 5. Session Lifecycles & Configuration

| Configuration Key | Default Value | Description |
| :--- | :--- | :--- |
| `EGO_SESSION_LIFESPAN` | `168h` (7 days) | Absolute maximum lifespan before re-authentication is enforced. |
| `EGO_SESSION_IDLE_TTL` | `24h` (1 day) | Inactivity timeout. Reset every time `/sessions/whoami` is called. |
| `EGO_SECURE_COOKIES` | `true` | Enforces `Secure` flag on HTTPS cookies. |
| `EGO_COOKIE_SAME_SITE` | `Lax` | CSRF protection default. |
