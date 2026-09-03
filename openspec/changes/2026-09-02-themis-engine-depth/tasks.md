# Tasks: Themis ABAC Policy Engine Depth (P6-S4)

- [x] 1. Expand Themis Zod Schemas
  - [x] 1.1 Add schemas for `policyVersion`, `policyFixture`, `testSuiteResult` in `console/src/lib/api/schemas/themis.ts`
  - [x] 1.2 Add schemas for `validationResult`, `dryRunResult`, and normalize evaluate response
  - [x] 1.3 Add unit tests in `console/src/lib/api/schemas/themis.test.ts`

- [x] 2. Implement Real Next.js BFF Routes for Themis
  - [x] 2.1 Update `console/src/app/api/themis/policies/route.ts` to proxy to Themis Go engine
  - [x] 2.2 Implement `console/src/app/api/themis/policies/[id]/route.ts` and `[id]/toggle/route.ts`
  - [x] 2.3 Implement `console/src/app/api/themis/eval/route.ts` with real evaluation and dry-run
  - [x] 2.4 Implement `console/src/app/api/themis/validate/route.ts`
  - [x] 2.5 Implement `console/src/app/api/themis/policies/[id]/versions/route.ts` and `rollback`
  - [x] 2.6 Implement `console/src/app/api/themis/policies/[id]/fixtures/route.ts` and `test-suite`
  - [x] 2.7 Add BFF tests in `console/src/app/api/themis/themis-bff.test.ts`

- [x] 3. Implement Themis UI Components & Playground
  - [x] 3.1 Enhance `dry-run-playground.tsx` with scratchpad, syntax validation, and rule evaluation tracer
  - [x] 3.2 Create Policy Test Suite runner modal/panel
  - [x] 3.3 Create Policy Version History and Rollback dialog
  - [x] 3.4 Wire components into `console/src/app/themis/page.tsx`
  - [x] 3.5 Add unit tests in `console/src/app/themis/themis-components.test.tsx`

- [x] 4. Verification & Roadmap Update
  - [x] 4.1 Run Vitest suite (`npm test` in `console`) and Next.js build validation (`npm run build`)
  - [x] 4.2 Update `docs/roadmap-control-plane.html` marking P6-S4 as DONE (7/7)
