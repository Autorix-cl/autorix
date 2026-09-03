# Tasks: Phase 7 Observability & Operations

## Milestone 1: Argus Metrics Aggregation (P7-S1-T6)
- [x] 1.1 Define `FleetMetricsSummary` and `EngineMetricSummary` in `argus/internal/core/metrics.go`
- [x] 1.2 Implement `handleGetMetricsSummary` in `argus/internal/transport/http/server.go`
- [x] 1.3 Add Go unit tests in `argus/internal/transport/http/server_test.go` and `internal/core/metrics_test.go`
- [x] 1.4 Mark `P7-S1` as DONE (6/6) in `docs/roadmap-control-plane.html`

## Milestone 2: Logging, Tracing & Correlation (P7-S2)
- [x] 2.1 Ensure correlation ID propagation across BFF proxy requests in `console/src/lib/api/proxy.ts`
- [x] 2.2 Create schemas for logs and distributed traces in `console/src/lib/api/schemas/observability.ts`
- [x] 2.3 Implement BFF routes for logs search and trace details (`/api/observability/logs`, `/api/observability/traces`)
- [x] 2.4 Build Console Log Viewer (`/observability/logs`) and Trace Waterfall (`/observability/traces`)

## Milestone 3: Console Dashboards, Alerts & SLOs (P7-S3)
- [x] 3.1 Adopt and configure shared charting components with themed tooltips
- [x] 3.2 Build Fleet Observability Dashboard (`/observability`) with RED metrics (Rate, Errors, Duration)
- [x] 3.3 Build Per-Engine Metric Dashboards (`/observability/engines/[type]`)
- [x] 3.4 Build Alert Rule Management (`/observability/alerts/rules`) & Notification Channels
- [x] 3.5 Build Active/Historical Alerts Monitor (`/observability/alerts`) & SLO Error Budgets (`/observability/slo`)
- [x] 3.6 Update `console/src/components/layout/sidebar.tsx` with Observability navigation links
- [x] 3.7 Run complete test suite and production build verification
