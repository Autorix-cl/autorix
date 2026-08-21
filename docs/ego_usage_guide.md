# Autorix Ego: Identity, Dynamic Traits & Session Engine Manual

**Autorix Ego** is an enterprise identity and user lifecycle engine inspired by Ory Kratos. It provides dynamic JSON Schema-driven identity traits, memory-hard Argon2id password hashing, RFC 6238 TOTP multi-factor authentication (MFA), recovery codes, and high-performance active session management.

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
  │  │ (Cookies & Bearer) │  │ & Recovery Codes Vault        │  │
  │  └─────────┬──────────┘  └───────────────┬───────────────┘  │
  └────────────┼─────────────────────────────┼──────────────────┘
               │                             │
               ▼                             ▼
                    [ PostgreSQL Database ]
                        (autorix_ego)
```

### 1.1 Dynamic JSON Schema Traits

Ego does not hardcode rigid user columns into relational tables. Instead, user attributes are modeled as **Dynamic Traits** validated against **JSON Schema (Draft 7)** definitions:
* Standard fields: `email`, `name`, `username`.
* Custom enterprise fields: `department`, `employee_id`, `cost_center`, `security_clearance`, `phone_number`.

---

## 🔐 2. Cryptographic Security Standards

### 2.1 Memory-Hard Argon2id Password Hashing

Ego strictly adheres to the OWASP password storage recommendations with Argon2id parameters configured to eliminate GPU/ASIC acceleration attacks:

| Parameter | Value | Security Rationale |
| :--- | :--- | :--- |
| **Memory Cost (`m`)** | `65536 KiB` (64 MB) | Forces high memory consumption per thread, rendering mass parallel GPU attacks impractical. |
| **Time Cost (`t`)** | `3 iterations` | Ensures optimal defense against specialized ASIC cracking rigs. |
| **Parallelism (`p`)** | `4 threads` | Leverages modern multi-core server hardware. |
| **Salt Length** | `16 bytes` | Cryptographically secure pseudo-random salt generated via CSPRNG. |
| **Key Length** | `32 bytes` | Resulting 256-bit derived cryptographic key. |

### 2.2 RFC 6238 TOTP MFA & Recovery Codes

* **TOTP Standard**: 6-digit codes refreshed every 30 seconds using HMAC-SHA1.
* **Backup Codes**: Single-use cryptographically random recovery codes generated during MFA enrollment.

---

## 📡 3. Complete REST API Reference

Ego runs on port `4433` and provides both self-service user endpoints and an administrative control API.

### 3.1 User Registration (`POST /self-service/registration`)

Registers a new identity, validates traits against the default schema, computes the Argon2id hash, and generates an active session.

* **Method**: `POST`
* **Path**: `/self-service/registration`
* **Headers**: `Content-Type: application/json`

#### Request Body
```json
{
  "traits": {
    "email": "alice@enterprise.corp",
    "name": "Alice Smith",
    "department": "Engineering"
  },
  "password": "SuperSecurePassword#2026"
}
```

#### Response (`201 Created`)
```json
{
  "identity": {
    "id": "c1a2b3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
    "schema_id": "default",
    "state": "active",
    "traits": {
      "email": "alice@enterprise.corp",
      "name": "Alice Smith",
      "department": "Engineering"
    },
    "created_at": "2026-08-20T08:00:00Z"
  },
  "session": {
    "id": "s1a2b3c4-d5e6-7f8a-9b0c-1d2e3f4a5b6c",
    "identity_id": "c1a2b3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
    "token": "ast_9f8e7d6c5b4a...",
    "expires_at": "2026-08-27T08:00:00Z"
  }
}
```
*Note: Also sets the `autorix_session_token` HTTP-only secure cookie.*

---

### 3.2 State-Machine Flow API (Ory Kratos Parity)

#### Initialize Registration Flow (`GET /self-service/registration/browser`)
Initializes a new registration flow with a unique `flow_id`, anti-CSRF token, and dynamic UI nodes.

* **Method**: `GET`
* **Path**: `/self-service/registration/browser`

##### Response (`200 OK`)
```json
{
  "id": "f8a1b2c3-d4e5-6789-0123-456789abcdef",
  "type": "registration",
  "state": "choose_method",
  "csrf_token": "csrf_9f8e7d6c...",
  "expires_at": "2026-08-20T22:15:00Z",
  "ui_nodes": [
    { "type": "input", "group": "password", "attributes": { "name": "traits.email", "type": "email" } },
    { "type": "input", "group": "password", "attributes": { "name": "password", "type": "password" } },
    { "type": "input", "group": "webauthn", "attributes": { "name": "webauthn_register", "type": "button" } }
  ]
}
```

#### Fetch Flow Details (`GET /self-service/registration/flows?id=...`)
Retrieves the current state and UI nodes of an active flow.

* **Method**: `GET`
* **Path**: `/self-service/registration/flows?id=f8a1b2c3-d4e5-6789-0123-456789abcdef`

#### WebAuthn / Passkeys Registration (`POST /self-service/webauthn/registration/start`)
Initiates FIDO2 / WebAuthn biometric credential registration challenge.

* **Method**: `POST`
* **Path**: `/self-service/webauthn/registration/start`
* **Finish Endpoint**: `POST /self-service/webauthn/registration/finish`

---

### 3.2 User Login (`POST /self-service/login`)

Authenticates an identity via identifier (`email`, `username`) and password.

* **Method**: `POST`
* **Path**: `/self-service/login`

#### Request Body
```json
{
  "identifier": "alice@enterprise.corp",
  "password": "SuperSecurePassword#2026"
}
```

#### Response (`200 OK`)
```json
{
  "identity": {
    "id": "c1a2b3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
    "traits": { "email": "alice@enterprise.corp", "name": "Alice Smith" }
  },
  "session": {
    "id": "s1a2b3c4-d5e6-7f8a-9b0c-1d2e3f4a5b6c",
    "token": "ast_9f8e7d6c5b4a...",
    "expires_at": "2026-08-27T08:00:00Z"
  }
}
```

---

### 3.3 Validate Current Session (`GET /sessions/whoami`)

Resolves the caller identity from the `autorix_session_token` cookie or `Authorization: Bearer <token>` header.

* **Method**: `GET`
* **Path**: `/sessions/whoami`

#### Response (`200 OK`)
```json
{
  "id": "s1a2b3c4-d5e6-7f8a-9b0c-1d2e3f4a5b6c",
  "active": true,
  "authenticated_at": "2026-08-20T08:00:00Z",
  "expires_at": "2026-08-27T08:00:00Z",
  "identity": {
    "id": "c1a2b3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
    "traits": { "email": "alice@enterprise.corp" }
  }
}
```

---

### 3.4 User Logout (`POST /self-service/logout`)

Revokes the active session token and clears the session cookie.

* **Method**: `POST`
* **Path**: `/self-service/logout`

#### Response (`200 OK`)
```json
{
  "status": "logged_out"
}
```

---

### 3.5 Admin: Identity Management

- `GET /admin/identities`: Lists identities with query search (`?q=`), state filter (`?state=active`), schema filter (`?schema_id=`), trait filters (`?filter.traits.department=Finance`), and cursor pagination.
- `GET /admin/identities/{id}`: Returns full identity profile by UUID.
- `PATCH /admin/identities/{id}`: Updates identity traits, state (`active`, `locked`, `suspended`), or schema ID.
- `DELETE /admin/identities/{id}`: Permanently deletes an identity and all associated credentials and sessions.

---

### 3.6 Admin: Password Reset & Recovery Links

- `POST /admin/identities/{id}/credentials/reset-password`: Resets a user's password directly or generates a secure temporary password with `force_rotation: true`.
- `POST /admin/identities/{id}/recovery-link`: Generates a time-bound recovery token and link (`/self-service/recovery?token=...`).

#### Reset Password Request Body
```json
{
  "password": "NewTemporaryPassword#2026",
  "force_rotation": true
}
```

---

### 3.7 Admin: MFA & Session Management

- `GET /admin/identities/{id}/mfa`: Returns TOTP MFA status (`totp_enabled`, `confirmed`, `backup_codes_remaining`).
- `DELETE /admin/identities/{id}/mfa`: Revokes TOTP configuration for a user.
- `GET /admin/sessions`: Lists all active cluster sessions with pagination.
- `DELETE /admin/sessions/{id}`: Revokes a specific session.
- `GET /admin/identities/{id}/sessions`: Lists all active sessions for a specific user.
- `DELETE /admin/identities/{id}/sessions`: Revokes all sessions for a user (force logout across all devices).

---

### 3.8 Admin: Identity Schemas Management

- `GET /admin/schemas`: Lists all registered JSON Schema trait definitions.
- `POST /admin/schemas`: Registers a new JSON Schema.
- `GET /admin/schemas/{id}`: Retrieves a schema definition by ID.
- `PATCH /admin/schemas/{id}`: Updates a schema.
- `DELETE /admin/schemas/{id}`: Removes a schema definition.

---

## 🛠️ 4. Production Recipes

### Custom Enterprise Trait Schema Example
```json
{
  "id": "corp_employee_v1",
  "name": "Corporate Employee",
  "schema": {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "properties": {
      "email": { "type": "string", "format": "email" },
      "name": { "type": "string", "minLength": 2 },
      "department": { "type": "string", "enum": ["Engineering", "Finance", "Security"] },
      "employee_id": { "type": "string", "pattern": "^EMP-[0-9]{4}$" }
    },
    "required": ["email", "name", "department"]
  }
}
```
