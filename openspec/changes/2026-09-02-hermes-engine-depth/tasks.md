# Tasks: Hermes Enterprise SAML 2.0 & SCIM 2.0 Depth (P6-S7)

- [x] 1. Expand Hermes Zod Schemas
  - [x] 1.1 Add `certificateInfoSchema` and update `samlProviderSchema` with certificate metadata in `console/src/lib/api/schemas/hermes.ts`
  - [x] 1.2 Add `scimMemberSchema`, `scimGroupSchema`, `scimGroupListResponseSchema`
  - [x] 1.3 Add `scimSyncHistorySchema` and `scimSyncHistoryListSchema`
  - [x] 1.4 Add schema tests in `console/src/lib/api/schemas/hermes.test.ts`

- [x] 2. Implement Real Next.js BFF Routes for Hermes
  - [x] 2.1 Implement `console/src/app/api/enterprise/saml/[id]/route.ts` (GET, PATCH, DELETE)
  - [x] 2.2 Implement `console/src/app/api/enterprise/saml/[id]/enable/route.ts` and `disable/route.ts` (POST)
  - [x] 2.3 Implement `console/src/app/api/enterprise/scim/groups/route.ts` (GET, POST) and `[id]/route.ts` (GET, PUT, PATCH, DELETE)
  - [x] 2.4 Implement `console/src/app/api/enterprise/scim/sync/route.ts` (GET, POST)
  - [x] 2.5 Add BFF tests in `console/src/app/api/enterprise/hermes-bff.test.ts`

- [x] 3. Implement Enterprise UI Components & Wizards
  - [x] 3.1 Create SAML Provider Detail & Attribute Mapper Dialog (`saml-provider-sheet.tsx`)
  - [x] 3.2 Create SAML Connection Wizard (`saml-connection-wizard.tsx`) with metadata parsing and test assertion step
  - [x] 3.3 Create SCIM Directory & Sync Monitor tabbed panel (`scim-sync-monitor.tsx`)
  - [x] 3.4 Wire upgraded components into `console/src/app/enterprise/page.tsx`
  - [x] 3.5 Add component tests in `console/src/app/enterprise/hermes-components.test.tsx`

- [x] 4. Verification & Roadmap Update
  - [x] 4.1 Run Vitest suite (`npm test` in `console`) and Next.js build validation (`npm run build`)
  - [x] 4.2 Update `docs/roadmap-control-plane.html` marking P6-S7 as DONE (6/6)
