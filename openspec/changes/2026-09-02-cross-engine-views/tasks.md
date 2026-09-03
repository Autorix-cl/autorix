# Tasks: Phase 6 Cross-Engine Views (P6-S8)

- [x] 1. Define Cross-Engine Zod Schemas & Types
  - [x] 1.1 Create `console/src/lib/api/schemas/explorer.ts` (`UnifiedSubject`, `EffectiveAccessResult`, `RequestSimulationTrace`, `ConsistencyFinding`)
  - [x] 1.2 Add schema unit tests in `console/src/lib/api/schemas/explorer.test.ts`

- [x] 2. Implement Cross-Engine BFF Aggregation Routes
  - [x] 2.1 Implement `console/src/app/api/explorer/subject/[id]/route.ts` (Resolving Ego, Sessions, Nexus, Janus, Vulcan, Hermes)
  - [x] 2.2 Implement `console/src/app/api/explorer/effective-access/route.ts` (Evaluating Aegis, Nexus, Themis)
  - [x] 2.3 Implement `console/src/app/api/explorer/simulate/route.ts` (Tracing end-to-end request lifecycle)
  - [x] 2.4 Implement `console/src/app/api/explorer/consistency/route.ts` (Running cross-engine configuration anomaly sweep)
  - [x] 2.5 Add unit tests in `console/src/app/api/explorer/explorer-bff.test.ts`

- [x] 3. Build Explorer UI & Interactive Simulators
  - [x] 3.1 Create `console/src/app/explorer/unified-subject-view.tsx` (Incident response profile, credentials, relations, sessions)
  - [x] 3.2 Create `console/src/app/explorer/effective-access-explorer.tsx` (Interactive access matrix & engine attribution breakdown)
  - [x] 3.3 Create `console/src/app/explorer/request-simulator.tsx` (Live visual tracer showing packet flow through the Zero-Trust mesh)
  - [x] 3.4 Create `console/src/app/explorer/consistency-checker.tsx` (Findings list with severity badges and remediation guidance)
  - [x] 3.5 Create `console/src/app/explorer/page.tsx` integrating all 4 views with tabbed navigation
  - [x] 3.6 Update `console/src/components/layout/sidebar.tsx` with Explorer navigation link
  - [x] 3.7 Add component tests in `console/src/app/explorer/explorer-components.test.tsx`

- [x] 4. Verification & Roadmap Update
  - [x] 4.1 Run Vitest suite (`npm test` in `console`) and Next.js build validation (`npm run build`)
  - [x] 4.2 Update `docs/roadmap-control-plane.html` marking P6-S8 as DONE (4/4)
