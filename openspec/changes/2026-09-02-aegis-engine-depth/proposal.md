# Change: Aegis Engine Management Depth (P6-S5)

## Context
In Autorix, Aegis serves as the Zero-Trust Access Proxy (Policy Enforcement Point - PEP) inspired by Ory Oathkeeper. The Go backend for Aegis has already completed its Phase 6 requirements: Postgres persistence in `autorix_aegis`, explicit rule ordering, version snapshotting with rollback, handler catalogues (`GET /handlers`), and dry-run execution with step-by-step pipeline traces (`POST /rules/test-match`).

However, the Autorix Console (`console/src/app/proxy-rules`) still operates with shallow prototypes:
1. It only performs read-only listing and basic URI matching tests without deep pipeline step traces.
2. Missing Next.js BFF API routes for `GET /api/proxy-rules/[id]`, `PUT /api/proxy-rules/reorder`, `GET /api/proxy-rules/handlers`, `GET /api/proxy-rules/versions`, and `POST /api/proxy-rules/rollback/[version]`.
3. Outdated Zod schemas in `console/src/lib/api/schemas/aegis.ts` missing `strip_prefix`, `rewrite`, `order_idx`, `PipelineTrace`, and catalogue definitions.
4. Absence of UI for the 5-step Rule Builder wizard, rule reordering with shadowing detection, and version history rollback.

## Objective
Deliver complete administrative depth for Aegis in the Console:
1. Update Zod contract schemas in `console/src/lib/api/schemas/aegis.ts` to fully mirror Aegis Go engine models.
2. Implement missing BFF routes in `console/src/app/api/proxy-rules/**`.
3. Build the 5-step Rule Builder Wizard supporting match, authenticators, authorizer, mutators, and upstream path rewriting (`strip_prefix`, `rewrite`) dynamic config from the handler catalogue.
4. Implement rule reordering with shadowing detection warning when a broad rule prevents a subsequent specific rule from matching.
5. Provide version history view and 1-click rollback to prior rule snapshots.
6. Enhance the simulator to visualize the complete 5-stage dry-run pipeline trace.
7. Update `docs/roadmap-control-plane.html` marking P6-S5 tasks as DONE.

## Scope
- **In Scope**:
  - `console/src/lib/api/schemas/aegis.ts`: Add `order_idx`, `strip_prefix`, `rewrite`, `pipelineTraceSchema`, `handlerCatalogueSchema`, `ruleVersionSchema`, and `reorderRulesSchema`.
  - `console/src/app/api/proxy-rules/**`: Implement `[id]/route.ts` (GET), `reorder/route.ts` (PUT), `handlers/route.ts` (GET), `versions/route.ts` (GET), and `rollback/[version]/route.ts` (POST).
  - `console/src/app/proxy-rules/**`: Rule builder wizard sheet/dialog, reordering controls with shadowing detector, version rollback drawer, and pipeline trace simulator view.
  - `docs/roadmap-control-plane.html`: Mark P6-S5 tasks as DONE.
- **Out of Scope**:
  - Modifying Aegis Go microservice (its engine tests and endpoints are already passing and verified).

## Capabilities
1. **aegis-console-bff**: Next.js App Router BFF routes proxying all Aegis management operations cleanly with Zod validation.
2. **aegis-rule-management**: Operator UI for guided rule creation, rule reordering with shadowing analysis, version rollback, and pipeline trace debugging.
