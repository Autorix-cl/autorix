# Specification: Janus OAuth2 & OIDC Depth Management (P6-S2)

## Capability: Client Lifecycle & Secret Overlap
Operators can inspect, edit, delete, and rotate client credentials with an overlap window.

### Scenario: Rotate Client Secret with Overlap
- **Given** an existing OAuth2 client `client-api-gateway`
- **When** the operator invokes `POST /api/oauth2/clients/client-api-gateway/rotate-secret` with an overlap window of 24 hours
- **Then** the backend generates a new cryptographically secure secret
- **And** retains the previous secret hash until `previous_secret_expires_at`
- **And** returns the newly generated plain-text secret to be displayed once to the operator.

### Scenario: Update Client Metadata
- **Given** an existing OAuth2 client `client-frontend`
- **When** the operator submits a `PATCH /api/oauth2/clients/client-frontend` updating `redirect_uris` or `client_name`
- **Then** the record in Postgres is updated
- **And** the updated client model is returned.

---

## Capability: Token Introspection & Revocation
Operators can verify token validity according to RFC 7662 and revoke tokens according to RFC 7009.

### Scenario: Introspect Valid Access Token
- **Given** an issued JWT or opaque token
- **When** the operator posts the token to `POST /api/oauth2/tokens/introspect`
- **Then** the response status is 200 with `{ active: true, sub: "user-123", scope: "openid profile", exp: 1770000000 }`.

### Scenario: Revoke Active Token
- **Given** an active token
- **When** the operator posts the token to `POST /api/oauth2/tokens/revoke`
- **Then** the response status is 200 with `{ status: "revoked" }`
- **And** subsequent introspection yields `{ active: false }`.

---

## Capability: Scope & Claims Catalogue
Operators can browse, register, and delete OAuth2 scopes and their mapped OpenID claims.

### Scenario: List and Register Scopes
- **Given** default scopes `openid`, `profile`, `email`
- **When** the operator submits `POST /api/oauth2/scopes` with name `billing:read` and claims `["org_id", "billing_tier"]`
- **Then** the new scope is persisted in `oauth2_scopes`
- **And** it appears in the scopes catalogue for client selection.

---

## Capability: JWKS Key Rollover
Operators can trigger key rotation to generate a new signing key and review active keys in the JWKS keystore.

### Scenario: Rotate Active Key
- **Given** one active RSA/ECDSA signing key
- **When** the operator triggers `POST /api/oauth2/keys/rotate`
- **Then** a new primary signing key is generated
- **And** previous keys remain published in JWKS for verification until their grace expiration.
