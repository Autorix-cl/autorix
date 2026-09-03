# Proposal: Janus OAuth2 & OIDC Engine Depth (P6-S2)

## Context & Motivation
Janus is Autorix's OAuth2 and OpenID Connect engine. In Phase 6 (Engine Management Depth), Janus requires comprehensive administrative control over client lifecycles, client secret rotation with overlap windows, scope and claims catalogues, token introspection (RFC 7662), token revocation (RFC 7009), key rotation with rollover, and consent grants management.

While the Go backend in `janus/` already provides endpoints and Postgres persistence for these capabilities, the Next.js Console (`console/src/app/oauth2`) currently lacks the corresponding BFF routes, extended Zod validation schemas, interactive Token Inspector, Scope Catalogue editor, Key Rotation controls, and Client Secret Overlap management.

## Scope of Changes
1. **Zod Contract Extensions (`console/src/lib/api/schemas/oauth2.ts`)**:
   - Add client rotation fields (`previous_secret_expires_at`, `has_previous_secret`) to `oauth2ClientSchema`.
   - Define `rotateSecretResponseSchema`, `oauth2ScopeSchema`, `oauth2ScopeListSchema`, `createScopeRequestSchema`.
   - Define `introspectRequestSchema`, `introspectResponseSchema`, `revokeTokenRequestSchema`, `revokeTokenResponseSchema`.
   - Define `rotateKeysResponseSchema`, `oauth2GrantSchema`, `oauth2GrantListSchema`.
2. **Next.js BFF API Routes**:
   - `GET/PATCH/DELETE /api/oauth2/clients/[id]`: Retrieve, update, or remove client apps.
   - `POST /api/oauth2/clients/[id]/rotate-secret`: Issue a new secret with grace period overlap.
   - `GET/POST /api/oauth2/scopes`: Retrieve and register managed scopes.
   - `DELETE /api/oauth2/scopes/[name]`: Remove custom scopes.
   - `POST /api/oauth2/tokens/introspect`: Proxy RFC 7662 token introspection.
   - `POST /api/oauth2/tokens/revoke`: Proxy RFC 7009 token revocation.
   - `POST /api/oauth2/keys/rotate`: Trigger JWKS key rollover.
   - `GET /api/oauth2/grants`: List active user consents/grants.
3. **UI Components & Screens (`console/src/app/oauth2`)**:
   - **Token Inspector & Revocation Tab**: Paste a bearer token or access token to inspect active claims, expiration, issuer, audience, and trigger immediate revocation.
   - **Scope Catalogue Tab**: Manage standard and custom scopes with descriptions and claim mappings.
   - **Key Rollover & JWKS Tab**: Inspect current active signing keys, key age, algorithm, and trigger instant key rotation with rollover preview.
   - **Client Detail & Secret Rotation Modal**: Rotate client secret with configurable expiration overlap window.
   - **Grants & Consent Tab**: Inspect and revoke user consents per client.
4. **Testing & Verification**:
   - Unit tests for Zod schemas (`oauth2.test.ts`).
   - Route handler tests for all new BFF endpoints (`oauth2-bff.test.ts`).
   - Component tests for new OAuth2 panels (`oauth2-components.test.tsx`).
   - Next.js production build (`npm run build`) verification.
