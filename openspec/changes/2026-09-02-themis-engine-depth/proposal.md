# Change Proposal: Themis ABAC Policy Engine Depth (P6-S4)

## Context
Phase 6 specifies deep engine management for all core engines. For Themis (ABAC Policies with CEL/Rego), the Go engine already implements Postgres-backed persistence, versioning & rollbacks, dry-run evaluation, test fixtures & test suite execution, CEL validation, and evaluation tracing.

The Console needs:
1. Complete Zod schemas matching Themis models (`PolicyVersion`, `PolicyFixture`, `TestSuiteResult`, `ValidationResult`, `DryRunResult`).
2. Complete Next.js BFF proxy routes in `console/src/app/api/themis/` bridging the real Themis Go engine instead of in-memory mocks.
3. Enhanced UI in `console/src/app/themis/`:
   - Interactive Policy Playground with live scratchpad, CEL validation, and decision evaluation trace.
   - Test Suite Runner executing attached fixtures with pass/fail reporting.
   - Version History and Rollback dialog.
4. Full verification in Vitest, Next.js build, and roadmap update.

## Scope
- Expand `console/src/lib/api/schemas/themis.ts`
- Implement Next.js BFF proxy endpoints for Themis
- Enhance `dry-run-playground.tsx`, `policies-table.tsx`, and `themis/page.tsx`
- Add tests in `themis.test.ts`, `themis-bff.test.ts`, and component tests
- Update `docs/roadmap-control-plane.html` marking P6-S4 as DONE
