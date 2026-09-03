# Tasks: Aegis Engine Management Depth (P6-S5)

- [x] 1. Expand Aegis Zod Schemas
  - [x] 1.1 Update `upstreamConfigSchema` with `strip_prefix` and `rewrite` fields in `console/src/lib/api/schemas/aegis.ts`
  - [x] 1.2 Add `order_idx` to `ruleSchema`
  - [x] 1.3 Add `pipelineTraceSchema`, `pipelineTraceStepSchema`, `handlerCatalogueSchema`, `ruleVersionSchema`, and `reorderRulesRequestSchema`
  - [x] 1.4 Add unit tests for schemas in `console/src/lib/api/schemas/aegis.test.ts`

- [x] 2. Implement Missing Next.js BFF Routes
  - [x] 2.1 Add `GET` method to `console/src/app/api/proxy-rules/[id]/route.ts`
  - [x] 2.2 Create `console/src/app/api/proxy-rules/reorder/route.ts` with `PUT` handler
  - [x] 2.3 Create `console/src/app/api/proxy-rules/handlers/route.ts` with `GET` handler
  - [x] 2.4 Create `console/src/app/api/proxy-rules/versions/route.ts` with `GET` handler
  - [x] 2.5 Create `console/src/app/api/proxy-rules/rollback/[version]/route.ts` with `POST` handler

- [x] 3. Implement Shadowing Detection Utility & Tests
  - [x] 3.1 Create `console/src/app/proxy-rules/shadowing-detector.ts` analyzing rule ordering overlaps
  - [x] 3.2 Add comprehensive unit test suite in `console/src/app/proxy-rules/shadowing-detector.test.ts`

- [x] 4. Build UI Components
  - [x] 4.1 Create `console/src/app/proxy-rules/rule-builder-sheet.tsx` (5-step wizard with catalogue schemas and live JSON preview)
  - [x] 4.2 Create `console/src/app/proxy-rules/versions-dialog.tsx` (version history timeline with rollback action)
  - [x] 4.3 Update `console/src/app/proxy-rules/page.tsx` integrating reordering controls, shadowing warning badges, wizard sheet, and deep pipeline trace viewer
  - [x] 4.4 Add UI component unit tests in `console/src/app/proxy-rules/components.test.tsx`

- [x] 5. Verification & Documentation
  - [x] 5.1 Run Vitest tests (`npm test` in `console`) and Next.js build validation (`npm run build`)
  - [x] 5.2 Update `docs/roadmap-control-plane.html` marking P6-S5 tasks as DONE
