# Migrate JWT trust and private administration

This hardening closes unsigned JWT acceptance and separates Janus administration
from its public listener. It is **not** a banking-readiness certification or a
complete security assessment. Upgrade the engines, console, and deployment
configuration together; older clients using public `/admin/*` URLs will stop working.

## Quick path

1. Keep Janus public OAuth/OIDC on `:4444`. Send management and login/consent
   acceptance requests to private `:4445`. Standalone Janus binds administration
   to `127.0.0.1` by default; containers explicitly use `ADMIN_HOST=0.0.0.0`.
2. Configure Aegis trusted issuer, audience, and JWKS using the environment
   configuration below. Rule-supplied `jwks_url` values do not establish trust.
3. Configure console `JANUS_ADMIN_INTERNAL_URL` separately from
   `JANUS_INTERNAL_URL`. Only server-side callers may use the admin address.
4. Run `python3 scripts/ci/check_security_boundaries.py` and the engine/console
   tests before deploying. Verify from outside the deployment that Janus
   `/admin/clients` returns `404` on the public listener.

## Trust configuration

| Setting | Purpose |
| --- | --- |
| `JWT_ISSUER` | Exact expected `iss`; must match Janus `ISSUER_URL`. |
| `JWT_AUDIENCE` | Exact expected `aud`; defaults in supplied deployments match the issuer for current Janus machine-to-machine tokens. |
| `JWT_JWKS_URL` | Operator-controlled JWKS source, never selected from a token or request. |
| `JWT_ALLOW_INSECURE_JWKS` | Defaults to `false`; Compose explicitly enables HTTP only for local development. |
| `JANUS_INTERNAL_URL` | Console backend for public JWKS/introspection/revocation. |
| `JANUS_ADMIN_INTERNAL_URL` | Console backend for `/admin/*`; never a `NEXT_PUBLIC_*` setting. |

The Helm chart uses the issuer's HTTPS JWKS endpoint. Its certificate must be
trusted by Aegis, and the ingress must be reachable from the Aegis pod. Compose
uses an internal HTTP JWKS URL for local development only; do not copy that
transport configuration into a production trust boundary.

The current Janus grants do **not** share an API audience model: client-credentials
access tokens use the issuer as audience, while authorization-code access tokens
and ID tokens use the client ID. This deployment intentionally accepts only the
machine-to-machine audience. Do not change the audience to accept browser tokens
until resource audiences and access-token versus ID-token separation are designed
and tested. Signature verification alone does not solve token-type confusion.

## OAuth client binding

- Janus accepts `client_credentials` only for confidential clients that explicitly
  registered that grant type. Every requested scope must be in the client's scope
  allowlist; authentication alone does not authorize a grant or scope.
- Authorization requests require `response_type=code`, an exact registered
  `redirect_uri`, an authorized scope set, and PKCE `S256`.
- Authorization-code exchange requires the same authenticated client and exact
  `redirect_uri`. The database consumes a code only after those bindings match,
  so a rejected exchange cannot burn a legitimate code.

## Private administration

- Helm exposes a separate `ClusterIP` Janus admin Service and never references
  it from an Ingress. Compose publishes neither Janus `:4445` nor Aegis `:4456`.
- The console validates sessions through Argus before forwarding Janus or Aegis
  requests. Janus requires `oauth2:read`/`oauth2:write`; Aegis requires
  `proxy-rules:read`/`proxy-rules:write`. Explicit route permissions are also
  enforced. Operators and auditors cannot mutate either engine via the BFF; a
  nonempty fabricated cookie cannot reach client or routing-rule administration.
- Login/consent acceptance belongs in a trusted backend after authenticating the
  user. Do not let browser-provided `subject` values establish identity.
- `ClusterIP` and an unpublished Docker port are not workload authentication.
  Before production, enforce firewall/NetworkPolicy restrictions and authenticated
  internal transport so only trusted management and login services reach admin.
- For local debugging, use `docker compose exec` inside the relevant engine or a
  temporary authenticated Kubernetes port-forward. Do not restore public admin
  mappings to make old scripts work. The CI smoke script uses internal requests;
  it destroys its Compose volumes at cleanup and is for disposable CI stacks only.

The Helm chart enables ingress-default-deny NetworkPolicies for Autorix pods.
It admits only configured ingress-controller traffic to the public edge,
in-namespace Autorix traffic, and configured monitoring scrapes. Before rollout,
verify `networkPolicy.ingressController` and `networkPolicy.monitoring` selectors
against the target cluster; an incorrect selector can block legitimate traffic.

Janus denies browser cross-origin requests unless `CORS_ALLOWED_ORIGINS` is
explicitly configured. The Helm default permits only the configured Console
origin; add each approved browser client deliberately rather than using `*`.

## Remaining production blockers

Janus signing keys are persisted in PostgreSQL and initialization/rotation is
serialized with a transaction-scoped advisory lock. Restarts retain the signing
set and retired verification keys. Multi-instance rotation propagation and
disaster-recovery key escrow still require operational verification before a
high-availability production claim.

Introspection/revocation require a confidential client and enforce token
ownership. Protocol conformance, resource audiences, full administrative
authorization across other engines, database
credentials/TLS, tenant isolation, and operational recovery still need separate
verification and hardening. This change does not claim to close those areas.

## Verification scope

The config guard renders Helm and parses Compose to detect accidental admin
publication and mismatched JWT settings. Console tests cover routing and session
and role checks; engine tests cover cryptographic validation and public/admin
route separation. These are regression checks, not a live deployment pentest.
