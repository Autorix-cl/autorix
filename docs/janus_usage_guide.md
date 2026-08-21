# Autorix Janus: OAuth 2.0 & OpenID Connect Server Manual

**Autorix Janus** is an enterprise OAuth 2.0 and OpenID Connect (OIDC) identity provider engine inspired by Ory Hydra. It issues cryptographically signed RS256 JWT access tokens, manages OAuth2 client lifecycles, handles PKCE-protected authorization flows, and maintains automated, zero-downtime JWKS key rotations.

---

## 🏛️ 1. Architecture & Protocol Compliance

```text
       [ Client Application / SPA / M2M Service ]
                           │
                           ▼ (OAuth 2.0 / OIDC :4444)
       ┌────────────────────────────────────────────────────────┐
       │                     Autorix Janus                      │
       │                                                        │
       │  ┌─────────────────────┐  ┌─────────────────────────┐  │
       │  │ OIDC Discovery      │  │ RS256 Key Manager       │  │
       │  │ (/.well-known)      │  │ (Automated JWKS Vault)  │  │
       │  └──────────┬──────────┘  └────────────┬────────────┘  │
       │             │                          │               │
       │  ┌──────────▼──────────┐  ┌────────────▼────────────┐  │
       │  │ OAuth2 Engine       │──┼──│ Token Signer         │  │
       │  │ (AuthCode/PKCE/M2M) │  │  │ (JWT Claims & Scopes)│  │
       │  └──────────┬──────────┘  └────────────┬────────────┘  │
       └─────────────┼──────────────────────────┼───────────────┘
                     │                          │
                     ▼                          ▼
               [ PostgreSQL Database: autorix_janus ]
```

### Supported Standards & Specifications

* **RFC 6749**: The OAuth 2.0 Authorization Framework.
* **RFC 7636**: Proof Key for Code Exchange (PKCE) with mandatory `S256` code challenge method.
* **RFC 7517 / RFC 7518**: JSON Web Keys (JWKS) and RS256 digital signature algorithm.
* **RFC 7662**: OAuth 2.0 Token Introspection.
* **RFC 7009**: OAuth 2.0 Token Revocation.
* **OpenID Connect Core 1.0**: Standardized identity discovery, JWKS export, and ID tokens.

---

## 🔑 2. Cryptographic Key Management & JWKS Rotation

Janus signs JWT access tokens using **asymmetric RSA 2048-bit** key pairs. Public keys are published at `/.well-known/jwks.json`.

### 2.1 Viewing Public JWKS (`GET /.well-known/jwks.json`)

```bash
curl -s http://localhost:4444/.well-known/jwks.json
```

**Response (`200 OK`):**
```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "key_2026_q3_active",
      "n": "u1L7Z3...[Base64URL modulus]...",
      "e": "AQAB"
    }
  ]
}
```

### 2.2 Zero-Downtime Key Rotation (`POST /admin/keys/rotate`)

Rotating the active signing key keeps previous keys in the JWKS set so existing tokens remain valid until expiration:

```bash
curl -X POST http://localhost:4444/admin/keys/rotate
```

---

## 📡 3. Complete REST API Reference

Janus operates on port `4444`.

### 3.1 OIDC Discovery (`GET /.well-known/openid-configuration`)

Returns OIDC discovery metadata including endpoints, supported grant types, and signing algorithms.

---

### 3.2 Decoupled Authorization Flow & Challenges (Ory Hydra Parity)

#### 1. Authorization Endpoint (`GET /oauth2/auth`)
Initiates the Authorization Code flow. Instead of handling login directly, Janus generates a `login_challenge` and redirects the user-agent to the configured Login UI:

```text
HTTP/1.1 302 Found
Location: http://localhost:3000/login?login_challenge=c1a2b3d4-e5f6...
```

#### 2. Accept Login Request (`PUT /admin/oauth2/auth/requests/login/accept`)
Called by the Login UI after authenticating the user (via Ego or external IdP):

##### Request Body
```json
{
  "challenge": "c1a2b3d4-e5f6...",
  "subject": "usr_9f8e7d6c5b4a"
}
```

##### Response (`200 OK`)
```json
{
  "redirect_to": "http://localhost:4444/oauth2/auth?login_challenge=...&login_verifier=..."
}
```

#### 3. Accept Consent Request (`PUT /admin/oauth2/auth/requests/consent/accept`)
Called by the Consent UI to grant requested OAuth2 scopes:

##### Request Body
```json
{
  "challenge": "cc_1a2b3c4d...",
  "granted_scopes": ["openid", "profile", "email"]
}
```

##### Response (`200 OK`)
```json
{
  "redirect_to": "http://localhost:4444/oauth2/auth?login_challenge=...&consent_challenge=...&consent_verifier=..."
}
```

---

### 3.2 Token Issuance (`POST /oauth2/token`)

Issues access tokens for `client_credentials`, `authorization_code` (with PKCE), and `refresh_token` flows.

* **Method**: `POST`
* **Path**: `/oauth2/token`
* **Headers**: `Content-Type: application/x-www-form-urlencoded`
* **Authentication**: HTTP Basic Auth (`client_id:client_secret`) or Form parameters.

#### Flow A: Machine-to-Machine (Client Credentials)
```bash
curl -X POST http://localhost:4444/oauth2/token \
  -u "client_backend_billing:SecretKey#2026" \
  -d "grant_type=client_credentials&scope=billing:read billing:write"
```

**Response (`200 OK`):**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "billing:read billing:write"
}
```

#### Flow B: Authorization Code with PKCE
```bash
curl -X POST http://localhost:4444/oauth2/token \
  -d "grant_type=authorization_code" \
  -d "client_id=spa_analytics_app" \
  -d "code=spl_auth_code_9a8b7c..." \
  -d "code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
```

---

### 3.3 Token Introspection (`POST /oauth2/introspect`)

Validates an access token and returns its active state, scopes, expiration, and identity claims (RFC 7662).

* **Method**: `POST`
* **Path**: `/oauth2/introspect`
* **Headers**: `Content-Type: application/x-www-form-urlencoded`

#### Request
```bash
curl -X POST http://localhost:4444/oauth2/introspect \
  -d "token=eyJhbGciOiJSUzI1NiIs..."
```

#### Response (`200 OK`)
```json
{
  "active": true,
  "scope": "billing:read billing:write",
  "client_id": "client_backend_billing",
  "sub": "client_backend_billing",
  "exp": 1787234400,
  "iat": 1787230800,
  "iss": "http://localhost:4444",
  "token_type": "Bearer"
}
```

---

### 3.4 Token Revocation (`POST /oauth2/revoke`)

Revokes an access token or refresh token immediately (RFC 7009).

* **Method**: `POST`
* **Path**: `/oauth2/revoke`
* **Headers**: `Content-Type: application/x-www-form-urlencoded`

```bash
curl -X POST http://localhost:4444/oauth2/revoke \
  -d "token=eyJhbGciOiJSUzI1NiIs..." \
  -d "token_type_hint=access_token"
```

#### Response (`200 OK`)
```json
{}
```

---

### 3.5 Admin: OAuth2 Client Management

- `POST /admin/clients`: Register a new OAuth2 client.
- `GET /admin/clients`: List clients with cursor pagination.
- `GET /admin/clients/{id}`: Get client configuration.
- `PATCH /admin/clients/{id}`: Update redirect URIs, scopes, or grant types.
- `DELETE /admin/clients/{id}`: Delete an OAuth2 client.
- `POST /admin/clients/{id}/rotate-secret`: Rotates client secret with configurable rollover overlap period (`overlap_seconds`).

#### Create Client Request Body
```json
{
  "client_id": "analytics_dashboard_spa",
  "client_name": "Analytics Dashboard SPA",
  "client_secret": "OptionalSecretForConfidentialClient",
  "grant_types": ["authorization_code", "refresh_token"],
  "redirect_uris": ["https://analytics.enterprise.corp/callback"],
  "scopes": ["openid", "profile", "email", "analytics:read"],
  "is_public": true
}
```

---

### 3.6 Admin: Scopes Catalogue & Grants

- `GET /admin/scopes`: List registered system scopes.
- `POST /admin/scopes`: Register a new scope (`name`, `description`).
- `DELETE /admin/scopes/{name}`: Delete a scope.
- `GET /admin/grants`: Query issued authorization grants.

---

## 🛠️ 4. Production Recipes

### Zero-Downtime Secret Rotation Recipe

```bash
curl -X POST http://localhost:4444/admin/clients/client_backend_billing/rotate-secret \
  -H "Content-Type: application/json" \
  -d '{ "overlap_seconds": 86400 }'
```

* Both the previous secret and new secret remain valid for 24 hours (`86400s`), enabling continuous zero-downtime deployment of backend consumers.
