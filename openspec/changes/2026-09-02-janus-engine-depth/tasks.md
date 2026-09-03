# Tasks: Janus OAuth2 & OIDC Engine Depth (P6-S2)

- [x] 1. Expand OAuth2 Zod Schemas
  - [x] 1.1 Update `oauth2ClientSchema` with `previous_secret_expires_at` and `has_previous_secret` in `console/src/lib/api/schemas/oauth2.ts`
  - [x] 1.2 Define `rotateSecretResponseSchema`, `oauth2ScopeSchema`, `oauth2ScopeListSchema`, `createScopeRequestSchema`
  - [x] 1.3 Define `introspectRequestSchema`, `introspectResponseSchema`, `revokeTokenRequestSchema`, `revokeTokenResponseSchema`
  - [x] 1.4 Define `rotateKeysResponseSchema`, `oauth2GrantSchema`, `oauth2GrantListSchema`
  - [x] 1.5 Add schema unit tests in `console/src/lib/api/schemas/oauth2.test.ts`

- [x] 2. Implement Missing Next.js BFF Routes
  - [x] 2.1 Implement `GET`, `PATCH`, `DELETE` in `console/src/app/api/oauth2/clients/[id]/route.ts`
  - [x] 2.2 Implement `POST` in `console/src/app/api/oauth2/clients/[id]/rotate-secret/route.ts`
  - [x] 2.3 Implement `GET`, `POST` in `console/src/app/api/oauth2/scopes/route.ts` and `DELETE` in `console/src/app/api/oauth2/scopes/[name]/route.ts`
  - [x] 2.4 Implement `POST` in `console/src/app/api/oauth2/tokens/introspect/route.ts` and `console/src/app/api/oauth2/tokens/revoke/route.ts`
  - [x] 2.5 Implement `POST` in `console/src/app/api/oauth2/keys/rotate/route.ts`
  - [x] 2.6 Implement `GET` in `console/src/app/api/oauth2/grants/route.ts`
  - [x] 2.7 Add BFF unit tests in `console/src/app/api/oauth2/oauth2-bff.test.ts`

- [x] 3. Implement UI Components
  - [x] 3.1 Create `console/src/app/oauth2/token-inspector.tsx` (RFC 7662 inspection & RFC 7009 revocation)
  - [x] 3.2 Create `console/src/app/oauth2/scope-catalogue.tsx` (scope management with claims chips)
  - [x] 3.3 Create `console/src/app/oauth2/key-manager.tsx` (JWKS key viewer with rollover trigger)
  - [x] 3.4 Create `console/src/app/oauth2/client-detail-dialog.tsx` (client secret rotation with overlap duration)
  - [x] 3.5 Update `console/src/app/oauth2/page.tsx` integrating tabbed interface for Clients, Token Inspector, Scopes, Keys, and Grants
  - [x] 3.6 Add UI component unit tests in `console/src/app/oauth2/oauth2-components.test.tsx`

- [x] 4. Verification & Roadmap Update
  - [x] 4.1 Run Vitest tests (`npm test` in `console`) and Next.js build validation (`npm run build`)
  - [x] 4.2 Update `docs/roadmap-control-plane.html` marking P6-S2 tasks as DONE

