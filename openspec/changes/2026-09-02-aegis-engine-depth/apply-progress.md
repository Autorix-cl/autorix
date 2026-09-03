# Apply Progress: Aegis Engine Management Depth (P6-S5)

- **Status**: Completed
- **Applied Changes**:
  - `console/src/lib/api/schemas/aegis.ts`: Extended Zod schemas for path rewriting, order index, dry-run pipeline trace, handler catalogue, versions, and reorder requests.
  - `console/src/app/api/proxy-rules/[id]/route.ts`: Added `GET` route handler.
  - `console/src/app/api/proxy-rules/reorder/route.ts`: Added `PUT` handler.
  - `console/src/app/api/proxy-rules/handlers/route.ts`: Added `GET` handler.
  - `console/src/app/api/proxy-rules/versions/route.ts`: Added `GET` handler.
  - `console/src/app/api/proxy-rules/rollback/[version]/route.ts`: Added `POST` handler.
  - `console/src/app/proxy-rules/shadowing-detector.ts`: Heuristic rule shadowing detection utility.
  - `console/src/app/proxy-rules/rule-builder-sheet.tsx`: 5-step guided wizard for pipeline rules with live JSON preview.
  - `console/src/app/proxy-rules/versions-dialog.tsx`: Version history inspection and 1-click rollback dialog.
  - `console/src/app/proxy-rules/page.tsx`: Integrated ordering controls, shadowing badges, wizard sheet trigger, and dry-run pipeline trace visualization.
  - `docs/roadmap-control-plane.html`: Updated P6-S5 to DONE.
- **Tests Added**:
  - `console/src/lib/api/schemas/aegis.test.ts`
  - `console/src/app/api/proxy-rules/proxy-rules-bff.test.ts`
  - `console/src/app/proxy-rules/shadowing-detector.test.ts`
  - `console/src/app/proxy-rules/components.test.tsx`
