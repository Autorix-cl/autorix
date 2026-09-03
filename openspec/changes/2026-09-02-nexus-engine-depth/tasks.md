# Tasks: Nexus ReBAC Zanzibar Engine Depth (P6-S3)

- [x] 1. Expand Nexus Zod Schemas
  - [x] 1.1 Add `decisionTraceSchema` and update `checkResponseSchema` with `trace` in `console/src/lib/api/schemas/nexus.ts`
  - [x] 1.2 Define `expandTreeSchema`, `expandResponseSchema`, `lookupSubjectsResponseSchema`, `lookupResourcesResponseSchema`
  - [x] 1.3 Define `namespaceSchema`, `namespaceListSchema`, `caveatSchema`, `caveatListSchema`
  - [x] 1.4 Add schema unit tests in `console/src/lib/api/schemas/nexus.test.ts`

- [x] 2. Implement Real Next.js BFF Routes for Nexus
  - [x] 2.1 Update `console/src/app/api/nexus/check/route.ts` to proxy to Nexus `POST /check` with explanation support
  - [x] 2.2 Update `console/src/app/api/nexus/tuples/route.ts` to proxy to Nexus `GET/POST/DELETE /tuples`
  - [x] 2.3 Update `console/src/app/api/nexus/schema/route.ts` to fetch and store schemas via `/admin/namespaces`
  - [x] 2.4 Implement `console/src/app/api/nexus/expand/route.ts` and `console/src/app/api/nexus/lookup/route.ts`
  - [x] 2.5 Implement `console/src/app/api/nexus/caveats/route.ts`
  - [x] 2.6 Add BFF unit tests in `console/src/app/api/nexus/nexus-bff.test.ts`

- [x] 3. Implement Nexus UI Components & Tabbed Navigation
  - [x] 3.1 Enhance `CheckSimulator` with walkable decision tree visualization in `console/src/app/nexus/check-simulator.tsx`
  - [x] 3.2 Enhance `TupleGrid` with namespace filters and CSV/JSON bulk import modal in `console/src/app/nexus/tuple-grid.tsx`
  - [x] 3.3 Create `RelationshipGraph` in `console/src/app/nexus/relationship-graph.tsx`
  - [x] 3.4 Create `CaveatRegistry` in `console/src/app/nexus/caveat-registry.tsx`
  - [x] 3.5 Update `console/src/app/nexus/page.tsx` with tabs: Playground (Editor + Simulator), Tuples Browser, Relationship Graph, Caveats Registry
  - [x] 3.6 Add UI component unit tests in `console/src/app/nexus/nexus-components.test.tsx`

- [x] 4. Verification & Roadmap Update
  - [x] 4.1 Run Vitest suite (`npm test` in `console`) and Next.js build validation (`npm run build`)
  - [x] 4.2 Update `docs/roadmap-control-plane.html` marking P6-S3 tasks as DONE
