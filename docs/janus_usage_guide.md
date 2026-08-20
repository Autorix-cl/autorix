# Autorix Janus: OAuth 2.0 & OpenID Connect Server Manual

**Autorix Janus** is a high-performance OAuth 2.0 and OpenID Connect (OIDC) identity provider engine inspired by Ory Hydra. It issues cryptographically signed RS256 JWT access tokens, manages OAuth2 client lifecycles, handles PKCE-protected authorization flows, and maintains dynamic JWKS key rotations.

---

## 🏛️ 1. Architecture & Protocol Compliance

```text
       [ Client Application / SPA ]
                   │
                   ▼ (OAuth2 Authorization Code + PKCE S256)
       ┌───────────────────────┐
       │     Autorix Janus     │ (REST :4444)
       │                       │
       │  ┌─────────────────┐  │  ┌───────────────────────┐
       │  │ OIDC Discovery  │  │  │ RS256 Key Manager     │
       │  │ (/.well-known)  │  │  │ (Auto JWKS Rotation)  │
       │  └────────┬────────┘  │  └───────────┬───────────┘
       │           │           │              │
       │  ┌────────▼────────┐  │  ┌───────────▼───────────┐
       │  │ OAuth2 Engine   │──┼──│ Token Signer          │
       │  │ (AuthCode/M2M)  │  │  │ (JWT Claims & Scopes) │
       │  └────────┬────────┘  │  └───────────┬───────────┘
       └───────────┼───────────┴──────────────┼───────────┘
                   │                          │
                   ▼                          ▼
               [ PostgreSQL Database: autorix_janus ]
```

### Standards Supported

* **RFC 6749**: The OAuth 2.0 Authorization Framework.
* **RFC 7636**: Proof Key for Code Exchange (PKCE) with mandatory `S256` challenge method.
* **RFC 7517 / RFC 7518**: JSON Web Keys (JWK/JWKS) and RS256 signature algorithms.
* **RFC 7662**: OAuth 2.0 Token Introspection.
* **RFC 7009**: OAuth 2.0 Token Revocation.
* **OpenID Connect Core 1.0**: ID Token issuance with user profile claims.

---

## 🔑 2. Cryptographic Key Management & JWKS Rotation

Janus uses asymmetric **RSA 2048-bit** key pairs for signing JWTs. Public keys are published at `/.well-known/jwks.json`.

### 2.1 Viewing the Public JWKS

```bash
curl -s http://localhost:4444/.well-known/jwks.json
```

**Response:**
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

### 2.2 Rotating Signing Keys (Zero-Downtime)

To rotate cryptographic keys without invalidating existing tokens:

```bash
curl -X POST http://localhost:4444/admin/keys/rotate
```

Janus generates a new active key for future token issuance while retaining the previous key in the JWKS for grace-period verification.

---

## 🚀 3. Client Registration & Scopes Catalog

### 3.1 Creating an OAuth2 Client

```bash
curl -X POST http://localhost:4444/admin/clients \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "Analytics Dashboard SPA",
    "grant_types": ["authorization_code", "refresh_token"],
    "response_types": ["code"],
    "scope": "openid profile email analytics:read",
    "redirect_uris": [
      "https://analytics.enterprise.io/callback",
      "http://localhost:3000/callback"
    ],
    "token_endpoint_auth_method": "none",
    "require_pkce": true
  }'
```

**Response (`201 Created`):**
```json
{
  "client_id": "cli_9a8b7c6d5e4f",
  "client_name": "Analytics Dashboard SPA",
  "grant_types": ["authorization_code", "refresh_token"],
  "scope": "openid profile email analytics:read",
  "created_at": "2026-08-20T09:00:00Z"
}
```

### 3.2 Machine-to-Machine (M2M) Client Creation

```bash
curl -X POST http://localhost:4444/admin/clients \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "Billing Cron Daemon",
    "grant_types": ["client_credentials"],
    "response_types": ["token"],
    "scope": "billing:read billing:write",
    "token_endpoint_auth_method": "client_secret_post"
  }'
```

**Response (`201 Created`):**
```json
{
  "client_id": "cli_billing_m2m",
  "client_secret": "sec_01918a7b6c5d4e3f2a1b0c9d8e7f6a",
  "grant_types": ["client_credentials"],
  "scope": "billing:read billing:write"
}
```

### 3.3 Rotating Client Secrets

```bash
curl -X POST http://localhost:4444/admin/clients/cli_billing_m2m/rotate-secret
```

---

## ⚡ 4. Token Issuance & Grant Flows

### 4.1 Client Credentials Grant (M2M)

```bash
curl -X POST http://localhost:4444/oauth2/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=cli_billing_m2m&client_secret=sec_01918a7b6c5d4e3f2a1b0c9d8e7f6a&scope=billing:read+billing:write"
```

**Response (`200 OK`):**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleV8yMDI2X3EzX2FjdGl2ZSIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "billing:read billing:write"
}
```

### 4.2 Authorization Code Flow with PKCE S256 (SPA / Mobile)

1. **Client Generates PKCE Verifier and Challenge**:
   $$\text{Code Challenge} = \text{Base64URL}(\text{SHA-256}(\text{Code Verifier}))$$
2. **Redirect to Janus `/oauth2/auth`**:
   ```text
   http://localhost:4444/oauth2/auth?
     response_type=code&
     client_id=cli_9a8b7c6d5e4f&
     redirect_uri=https://analytics.enterprise.io/callback&
     scope=openid+profile+email&
     state=xyz123&
     code_challenge=E9Melhoa2OwvFrGMTJguCH5...&
     code_challenge_method=S256
   ```
3. **Exchanging Authorization Code for Tokens**:
   ```bash
   curl -X POST http://localhost:4444/oauth2/token \
     -H "Content-Type: application/x-www-form-urlencoded" \
     -d "grant_type=authorization_code&client_id=cli_9a8b7c6d5e4f&code=auth_code_12345&redirect_uri=https://analytics.enterprise.io/callback&code_verifier=high_entropy_verifier_string"
   ```

---

## 🔍 5. Token Introspection & Revocation

### 5.1 Introspect Token (RFC 7662)

Downstream APIs and proxies call this endpoint to check if an opaque or JWT token is active:

```bash
curl -X POST http://localhost:4444/oauth2/introspect \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=eyJhbGciOiJSUzI1Ni..."
```

**Response:**
```json
{
  "active": true,
  "scope": "billing:read billing:write",
  "client_id": "cli_billing_m2m",
  "sub": "cli_billing_m2m",
  "exp": 1787220000,
  "iat": 1787216400,
  "iss": "http://localhost:4444"
}
```

### 5.2 Revoke Token (RFC 7009)

```bash
curl -X POST http://localhost:4444/oauth2/revoke \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=eyJhbGciOiJSUzI1Ni...&client_id=cli_billing_m2m&client_secret=sec_..."
```
