# Manage OAuth 2.0 & OpenID Connect with Autorix Janus

**Autorix Janus** is an enterprise OAuth 2.0 and OpenID Connect (OIDC) identity provider engine inspired by Ory Hydra. It issues cryptographically signed RS256 JWT access tokens, manages OAuth2 client lifecycles, handles PKCE-protected authorization flows, and maintains automated, zero-downtime JWKS key rotations.

## Quick path

Get an access token for a machine-to-machine service using the Client Credentials flow:

```bash
curl -X POST http://localhost:4444/oauth2/token \
  -u "client_backend_billing:SecretKey#2026" \
  -d "grant_type=client_credentials&scope=billing:read billing:write"
```

## Details

### Architecture & Protocol Compliance

Janus complies with RFC 6749 (OAuth 2.0), RFC 7636 (PKCE), RFC 7517/7518 (JWKS), RFC 7662 (Token Introspection), RFC 7009 (Token Revocation), and OpenID Connect Core 1.0.

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

### Cryptographic Key Management

Janus signs JWT access tokens using asymmetric RSA 2048-bit key pairs. Key material is persisted in PostgreSQL; startup atomically loads the shared key set, and rotation retains prior public keys for verification rollover.
- **View Public JWKS:** `GET /.well-known/jwks.json`
- **Rotate Keys:** `POST /admin/keys/rotate` (zero-downtime; keeps previous keys valid until expiration)
- **Rotate Client Secrets:** `POST /admin/clients/{id}/rotate-secret` (configurable overlap period for continuous deployment)

Access tokens carry `token_use: access_token`; ID tokens carry `token_use: id_token` and are issued only when `openid` was granted. Aegis requires the access-token marker, so an ID token cannot be replayed to an API.

### Decoupled Authorization Flow & Challenges

Janus delegates user authentication and consent to your own UI:
1. **Initiate:** `GET /oauth2/auth` generates a `login_challenge` and redirects to the Login UI.
2. **Accept Login:** UI calls `PUT /admin/oauth2/auth/requests/login/accept` with user context.
3. **Accept Consent:** UI calls `PUT /admin/oauth2/auth/requests/consent/accept` with granted scopes.

### Refresh Token Rotation

Janus issues a refresh token only after an authorization-code exchange when **both** conditions are met: the registered client includes `refresh_token` in `grant_types`, and the approved grant includes `offline_access`. The token is a 256-bit, URL-safe opaque value; Janus stores only its SHA-256 hash.

Use the token endpoint with the same client authentication that owns the refresh token:

```bash
curl -X POST http://localhost:4444/oauth2/token \
  -u "client_id:client_secret" \
  -d "grant_type=refresh_token&refresh_token=REDACTED"
```

Every successful refresh rotates the token. The replacement preserves the original client, subject, granted scopes, and RFC 8707 resource; requests cannot change `scope` or `resource` during refresh. Reusing a predecessor revokes the complete token family and returns `invalid_grant`, so clients must atomically replace their stored token after every response.

`REFRESH_TOKEN_TTL` configures the lifetime and defaults to `720h` (30 days). Set a positive Go duration appropriate for the deployment; changing it affects newly issued tokens only.

### Public API Reference (Port `4444`)

- **Token Issuance:** `POST /oauth2/token`
- **Token Introspection:** `POST /oauth2/introspect`
- **Token Revocation:** `POST /oauth2/revoke`
- **OIDC Discovery:** `GET /.well-known/openid-configuration`

`/oauth2/introspect` and `/oauth2/revoke` require authentication by a registered **confidential** client (HTTP Basic or form credentials). A client can only inspect or revoke its own tokens; unauthorized token ownership returns the RFC-compatible inactive/empty response.

### Private Admin API Reference (Port `4445`)

Never expose this listener publicly. Login and consent acceptance also use this
listener from a trusted backend, not directly from the browser. See the
[security migration guide](./security_boundary_migration.md).

- **Admin Clients:** `POST /admin/clients`, `GET /admin/clients`, `GET /admin/clients/{id}`, `PATCH /admin/clients/{id}`, `DELETE /admin/clients/{id}`
- **Admin Scopes:** `POST /admin/scopes`, `GET /admin/scopes`, `DELETE /admin/scopes/{name}`
- **Admin Grants:** `GET /admin/grants`

## Checklist

- [ ] Connect Janus to a PostgreSQL database (`autorix_janus`).
- [ ] Generate and verify the JWKS public keys at `/.well-known/jwks.json`.
- [ ] Configure the Login and Consent UIs for decoupled authorization.
- [ ] Register OAuth2 clients and system scopes via the Admin API.
- [ ] Test authorization code and client credentials grant flows.

## Next step

Learn how to configure fine-grained permissions and ABAC by reading the [Autorix Nexus Usage Guide](./nexus_usage_guide.md).
