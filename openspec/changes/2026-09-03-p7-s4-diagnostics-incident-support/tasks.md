# Tasks: Phase 7 Spec 4 Diagnostics & Incident Support (P7-S4)

- [x] 1. Define Diagnostics Zod Schemas & Types
  - [x] 1.1 Create `console/src/lib/api/schemas/diagnostics.ts`
  - [x] 1.2 Add schema unit tests in `console/src/lib/api/schemas/diagnostics.test.ts`

- [x] 2. Implement Diagnostics BFF Route Handlers
  - [x] 2.1 Implement `console/src/app/api/diagnostics/probe/route.ts` (P7-S4-T1)
  - [x] 2.2 Implement `console/src/app/api/diagnostics/drift/route.ts` (P7-S4-T2)
  - [x] 2.3 Implement `console/src/app/api/diagnostics/migrations/route.ts` (P7-S4-T3)
  - [x] 2.4 Implement `console/src/app/api/diagnostics/timeline/route.ts` (P7-S4-T4)
  - [x] 2.5 Implement `console/src/app/api/diagnostics/bundle/route.ts` (P7-S4-T5)
  - [x] 2.6 Add unit tests in `console/src/app/api/diagnostics/diagnostics-bff.test.ts`

- [x] 3. Build UI Components & Diagnostics Tab
  - [x] 3.1 Create `console/src/app/observability/diagnostics-manager.tsx`
  - [x] 3.2 Update `console/src/app/observability/page.tsx` with Diagnostics tab
  - [x] 3.3 Add component unit tests in `console/src/app/observability/diagnostics-components.test.tsx`

- [x] 4. Verification & Roadmap Update
  - [x] 4.1 Run Vitest tests (`npm test` in `console/`) - 75 test suites / 343 tests passed
  - [x] 4.2 Run Next.js production build (`npm run build` in `console/`) - Compiled cleanly
  - [x] 4.3 Update `docs/roadmap-control-plane.html` marking P7-S4 as DONE (5/5) and Phase 7 as fully completed (23/23 tasks)
