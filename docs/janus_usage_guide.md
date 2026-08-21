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

Janus signs JWT access tokens using asymmetric RSA 2048-bit key pairs. 
- **View Public JWKS:** `GET /.well-known/jwks.json`
- **Rotate Keys:** `POST /admin/keys/rotate` (zero-downtime; keeps previous keys valid until expiration)
- **Rotate Client Secrets:** `POST /admin/clients/{id}/rotate-secret` (configurable overlap period for continuous deployment)

### Decoupled Authorization Flow & Challenges

Janus delegates user authentication and consent to your own UI:
1. **Initiate:** `GET /oauth2/auth` generates a `login_challenge` and redirects to the Login UI.
2. **Accept Login:** UI calls `PUT /admin/oauth2/auth/requests/login/accept` with user context.
3. **Accept Consent:** UI calls `PUT /admin/oauth2/auth/requests/consent/accept` with granted scopes.

### REST API Reference (Port `4444`)

- **Token Issuance:** `POST /oauth2/token`
- **Token Introspection:** `POST /oauth2/introspect`
- **Token Revocation:** `POST /oauth2/revoke`
- **OIDC Discovery:** `GET /.well-known/openid-configuration`
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
