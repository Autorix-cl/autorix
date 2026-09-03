# Proposal: Phase 7 Spec 4 - Diagnostics & Incident Support (P7-S4)

## Context
Spec P7-S4 completes Phase 7 ("Observability & Operations") by delivering active incident diagnostic capabilities to the Autorix Control Plane:
- P7-S4-T1: Connectivity Troubleshooter (DNS, TCP, TLS, Application response live probes).
- P7-S4-T2: Configuration Drift Detection across replicated engine instances.
- P7-S4-T3: Database Migration Status across all 7 engines.
- P7-S4-T4: Change Correlation Timeline overlaying changes onto error spikes.
- P7-S4-T5: One-click Diagnostic Bundle Export for support escalation.

## Design
1. **Schemas (`console/src/lib/api/schemas/diagnostics.ts`)**:
   - `connectivityProbeResultSchema`: Granular 4-stage network breakdown.
   - `configDriftFindingSchema`: Instance divergence detection.
   - `migrationStatusSchema`: Database migration version checker.
   - `changeTimelineEventSchema`: Correlation events with error spikes.
   - `diagnosticBundleSchema`: Archive metadata and export payload.
2. **BFF Handlers (`console/src/app/api/diagnostics/`)**:
   - `/probe`, `/drift`, `/migrations`, `/timeline`, `/bundle`.
3. **UI Components (`console/src/app/observability/diagnostics-manager.tsx`)**:
   - Integrated into the Observability hub (`/observability`) with interactive execution, live status indicators, and export actions.
