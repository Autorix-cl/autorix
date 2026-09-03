# Design: Aegis Engine Management Depth

## Architecture & Integration Flow

The Aegis Go engine already exposes the required admin endpoints on its admin port (default `:4456` or `/admin/*` via `/rules`, `/handlers`, etc.):
- `GET /rules`: List all rules ordered by `order_idx`
- `POST /rules`: Create new rule
- `GET /rules/{id}`: Fetch single rule
- `PUT /rules/{id}`: Update rule
- `DELETE /rules/{id}`: Remove rule
- `PUT /rules/reorder`: Persist rule ordering
- `GET /rules/versions`: List rule snapshots
- `POST /rules/rollback/{version}`: Rollback to version snapshot
- `GET /handlers`: Handlers catalogue (authenticators, authorizers, mutators metadata and schemas)
- `POST /rules/test-match`: Dry-run pipeline simulator returning `PipelineTrace`

### Next.js BFF Layer
We will add standard BFF routes using `proxyRequest("aegis", ...)`:
1. `console/src/app/api/proxy-rules/[id]/route.ts`: Add `GET` handler.
2. `console/src/app/api/proxy-rules/reorder/route.ts`: `PUT` handler proxying to `/rules/reorder`.
3. `console/src/app/api/proxy-rules/handlers/route.ts`: `GET` handler proxying to `/handlers`.
4. `console/src/app/api/proxy-rules/versions/route.ts`: `GET` handler proxying to `/rules/versions`.
5. `console/src/app/api/proxy-rules/rollback/[version]/route.ts`: `POST` handler proxying to `/rules/rollback/${version}`.

### Schemas (`console/src/lib/api/schemas/aegis.ts`)
Extend Zod schemas:
- `upstreamConfigSchema`: Add optional `strip_prefix: z.string().optional()` and `rewrite: z.string().optional()`.
- `ruleSchema`: Add optional `order_idx: z.number().optional()`.
- `pipelineTraceStepSchema`: Mirror `core.PipelineTraceStep` (`stage`, `handler`, `status`, `details`, `allowed`, `session`, `mutated_headers`, `target_url`, `error`).
- `pipelineTraceSchema`: Mirror `core.PipelineTrace` (`matched_rule_id`, `steps`, `final_verdict`, `error`).
- `testMatchResponseSchema`: Support both legacy `{ matched: boolean, rule?: Rule }` and enhanced trace `{ matched: boolean, rule?: Rule, trace?: PipelineTrace }`.
- `handlerInfoSchema` & `handlerCatalogueSchema`: Mirror `core.HandlerCatalogue`.
- `ruleVersionSchema` & `ruleVersionListSchema`: Mirror `core.RuleVersion`.
- `reorderRulesRequestSchema`: `{ ids: string[] }`.

### Console UI Components
1. `rule-builder-sheet.tsx`:
   - 5-step wizard in a Sheet / Dialog component.
   - Dynamic handler picker querying `/api/proxy-rules/handlers`.
   - Upstream configuration supporting URL, strip prefix, and regex rewrite rules.
   - Live preview panel with synthesized JSON rule structure.
2. `shadowing-detector.ts`:
   - Heuristic utility function `detectShadowedRules(rules: Rule[]): Map<string, string>`
   - Detects if an earlier rule `R_prev` has an identical or superset HTTP method list and a broader URL glob/regex matching all paths that a later rule `R_curr` would match.
3. `versions-dialog.tsx`:
   - Modal displaying snapshot history from `/api/proxy-rules/versions`.
   - Displays snapshot timestamp, rule count, and rollback trigger with confirmation.
4. `page.tsx`:
   - Enhanced rules table with Up/Down reordering triggers (and drag handle).
   - Amber warning badges on shadowed rules with descriptive tooltip/popover.
   - Trigger buttons for "New Rule" (opens wizard) and "Version History" (opens versions dialog).
   - Upgraded Pipeline Simulator section rendering stage-by-stage pipeline steps.
