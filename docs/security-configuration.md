# Security configuration

Start from these controls before integrating an application. They are a
baseline, not a certification or a complete threat model.

## JWT trust at Aegis

Configure exactly one trusted key source: `JWT_JWKS_URL` **or**
`JWT_PUBLIC_KEY_FILE`. Set `JWT_ISSUER` and `JWT_AUDIENCE` to the exact values
your API accepts. Production JWKS endpoints must use HTTPS;
`JWT_ALLOW_INSECURE_JWKS=true` is only for the local Compose stack.

Aegis accepts access tokens only. Keep issuer, audience, and resource
registration under operator control; a request or proxy rule must never select
the JWKS URL.

## Administration and Console

- Keep Janus public OAuth/OIDC traffic on `:4444`; send `/admin/*` only through
  its private listener.
- Keep Aegis administration private. Do not publish `:4456` or permit broad
  browser CORS origins.
- The Console server-side BFF uses internal addresses. Never expose
  `JANUS_ADMIN_INTERNAL_URL` as a `NEXT_PUBLIC_*` setting.
- Register exact redirect URIs, grant types, scopes, and resource audiences per
  OAuth client. Require PKCE S256 for authorization-code clients.

## Secrets and deployment identity

Use a secret manager and workload identity for database URLs, signing material,
and bootstrap secrets. Pin production images by verified OCI digest, require
TLS to PostgreSQL, and rotate credentials with tested rollback procedures.

See [the security boundary migration](./security_boundary_migration.md) for
configuration details and verification commands.
