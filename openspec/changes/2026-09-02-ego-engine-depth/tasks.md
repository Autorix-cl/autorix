# Tasks: Ego Identities, Sessions & Trait Management Depth (P6-S1)

- [x] 1. Expand Ego Zod Schemas
  - [x] 1.1 Add schemas for `session`, `sessionList`, `credential`, `credentialList`
  - [x] 1.2 Add schemas for `mfaStatus`, `recoveryLink`, `traitSchema`, `bulkImportResult`
  - [x] 1.3 Add unit tests in `console/src/lib/api/schemas/identity.test.ts`

- [x] 2. Implement Real Next.js BFF Routes for Ego
  - [x] 2.1 Implement `console/src/app/api/identities/[id]/route.ts` (GET, PATCH, DELETE)
  - [x] 2.2 Implement `console/src/app/api/identities/[id]/sessions/route.ts` (GET, DELETE)
  - [x] 2.3 Implement `console/src/app/api/sessions/route.ts` and `console/src/app/api/sessions/[id]/route.ts`
  - [x] 2.4 Implement `console/src/app/api/identities/[id]/credentials/route.ts`, `reset-password`, and `recovery-link`
  - [x] 2.5 Implement `console/src/app/api/identities/[id]/mfa/route.ts` (GET, DELETE)
  - [x] 2.6 Implement `console/src/app/api/identities/schemas/route.ts` (GET, POST)
  - [x] 2.7 Add BFF tests in `console/src/app/api/identities/identities-bff.test.ts`

- [x] 3. Enhance Identities UI Components
  - [x] 3.1 Wire `IdentitySheet` with live session revocation, reset password, recovery link, and MFA reset
  - [x] 3.2 Add Bulk Import / Export Modal with dry-run CSV validation in `console/src/app/identities/`
  - [x] 3.3 Add unit tests in `console/src/app/identities/identity-sheet.test.tsx` and bulk import tests

- [x] 4. Verification & Roadmap Update
  - [x] 4.1 Run Vitest suite (`npm test` in `console`) and Next.js build validation (`npm run build`)
  - [x] 4.2 Update `docs/roadmap-control-plane.html` marking P6-S1 as DONE (8/8)
