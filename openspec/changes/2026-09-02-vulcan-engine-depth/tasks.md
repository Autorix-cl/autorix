# Tasks: Vulcan API Keys, Attenuation & Macaroons Depth (P6-S6)

- [x] 1. Expand Vulcan Zod Schemas
  - [x] 1.1 Update `apiKeySchema` with `call_count`, `last_source_ip`, `grace_period_expires_at`, `description`
  - [x] 1.2 Add `rotateKeyResponseSchema`, `scopeSchema`, `scopeListSchema`, `verifyKeyResponseSchema`
  - [x] 1.3 Add schema tests in `console/src/lib/api/schemas/vulcan.test.ts`

- [x] 2. Implement Real Next.js BFF Routes for Vulcan
  - [x] 2.1 Implement `console/src/app/api/vulcan/keys/[id]/route.ts` (GET, PATCH)
  - [x] 2.2 Implement `console/src/app/api/vulcan/keys/[id]/rotate/route.ts` (POST)
  - [x] 2.3 Implement `console/src/app/api/vulcan/scopes/route.ts` (GET, POST) and `[name]/route.ts` (DELETE)
  - [x] 2.4 Implement `console/src/app/api/vulcan/verify/route.ts` (POST)
  - [x] 2.5 Add BFF tests in `console/src/app/api/vulcan/vulcan-bff.test.ts`

- [x] 3. Enhance Vulcan UI Components
  - [x] 3.1 Update `KeysTable` with call counts, stale key warnings, and key rotation dialog
  - [x] 3.2 Create `ScopeCatalogSheet` component for managing platform scopes
  - [x] 3.3 Enhance `AttenuationStudio` with visual capability narrowing chain
  - [x] 3.4 Create `MacaroonInspector` component to paste and inspect tokens with live verification
  - [x] 3.5 Wire into `console/src/app/vulcan/page.tsx`
  - [x] 3.6 Add component tests in `console/src/app/vulcan/vulcan-components.test.tsx`

- [x] 4. Verification & Roadmap Update
  - [x] 4.1 Run Vitest suite (`npm test` in `console`) and Next.js build validation (`npm run build`)
  - [x] 4.2 Update `docs/roadmap-control-plane.html` marking P6-S6 as DONE (6/6)
