# Change Proposal: Phase 7 Observability & Operations (P7-S1, P7-S2, P7-S3)

## Context
Phase 7 ("Observability & Operations") makes the Autorix fleet measurable, debuggable, and alertable from the unified control plane.

Existing state:
- `P7-S1`: Metrics instrumentation (5/6 tasks done; P7-S1-T6 "Argus metrics aggregation" pending).
- `P7-S2`: Logging and tracing (Correlation ID propagation, OpenTelemetry tracing, log aggregation, log search viewer, trace waterfall).
- `P7-S3`: Dashboards and alerting in the console (Charting foundation, fleet dashboard, per-engine dashboards, alert rule management, notification channels, alert lifecycle, SLO definitions).

## Scope
1. **Argus Fleet Metrics Aggregator (P7-S1-T6)**:
   - Expose `GET /v1/metrics/summary` in Argus HTTP transport.
   - Aggregate instance status, QPS, error rate, p50/p95/p99 latency, and authorization allow/deny ratios per engine and across the entire fleet.
2. **Correlation ID & OpenTelemetry Tracing (P7-S2-T1, P7-S2-T2, P7-S2-T4, P7-S2-T5)**:
   - Structured correlation ID propagation (`X-Request-Id` / `X-Trace-Id`) across the BFF and engine hops.
   - Trace waterfalls and span viewers in Console.
3. **Observability Dashboards & Alerts (P7-S3-T1, P7-S3-T2, P7-S3-T3, P7-S3-T4, P7-S3-T5, P7-S3-T6, P7-S3-T7)**:
   - Modern fleet observability screen (`/observability`) with real-time metrics, per-engine charts, and SLO error budgets.
   - Integrated alert management (`/observability/alerts` & `/observability/alerts/rules`).
