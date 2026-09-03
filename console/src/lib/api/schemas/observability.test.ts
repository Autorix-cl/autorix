import { describe, it, expect } from "vitest";
import {
  fleetMetricsSummarySchema,
  logEntryListSchema,
  traceDetailSchema,
  alertRuleListSchema,
  alertEventListSchema,
  sloListSchema,
} from "./observability";

describe("Observability Zod Schemas", () => {
  it("validates fleetMetricsSummarySchema", () => {
    const summary = {
      timestamp: "2026-09-03T10:00:00Z",
      total_engines: 7,
      total_instances: 14,
      healthy_instances: 14,
      requests_total: 150000,
      fleet_qps: 185.4,
      fleet_error_rate: 0.008,
      fleet_latency_p95_ms: 4.2,
      auth_decisions_total: 80000,
      auth_allow_rate: 0.985,
      engines: [
        {
          engine_type: "nexus",
          status: "healthy",
          instance_count: 2,
          requests_total: 45000,
          requests_per_second: 37.5,
          error_rate: 0.005,
          latency_p50_ms: 2.1,
          latency_p95_ms: 5.2,
          latency_p99_ms: 12.0,
          auth_decisions_total: 45000,
          auth_allow_rate: 0.99,
        },
      ],
    };
    expect(fleetMetricsSummarySchema.safeParse(summary).success).toBe(true);
  });

  it("validates logEntryListSchema", () => {
    const logs = [
      {
        id: "log_1",
        timestamp: "2026-09-03T10:00:01Z",
        engine: "aegis",
        level: "info",
        message: "Proxy request routed to upstream",
        request_id: "req_123",
        correlation_id: "corr_123",
        attributes: { status: 200, latency_ms: 1.4 },
      },
    ];
    expect(logEntryListSchema.safeParse(logs).success).toBe(true);
  });

  it("validates traceDetailSchema", () => {
    const trace = {
      trace_id: "tr_abc",
      root_operation: "HTTP GET /api/v1/projects",
      root_engine: "aegis",
      total_duration_ms: 6.8,
      timestamp: "2026-09-03T10:00:00Z",
      status: "ok",
      spans: [
        {
          span_id: "sp_root",
          trace_id: "tr_abc",
          engine: "aegis",
          operation: "Ingress Router",
          status: "ok",
          start_time_ms: 0,
          duration_ms: 6.8,
          attributes: {},
        },
        {
          span_id: "sp_nexus",
          parent_span_id: "sp_root",
          trace_id: "tr_abc",
          engine: "nexus",
          operation: "CheckRelation",
          status: "ok",
          start_time_ms: 1.2,
          duration_ms: 2.3,
          attributes: { namespace: "project" },
        },
      ],
    };
    expect(traceDetailSchema.safeParse(trace).success).toBe(true);
  });

  it("validates alertRuleListSchema and alertEventListSchema", () => {
    const rules = [
      {
        id: "rule_high_error",
        name: "High Engine Error Rate",
        engine_type: "aegis",
        severity: "critical",
        metric: "http_requests_error_ratio",
        threshold: 0.05,
        operator: "gt",
        duration: "2m",
        enabled: true,
      },
    ];
    expect(alertRuleListSchema.safeParse(rules).success).toBe(true);

    const events = [
      {
        id: "evt_1",
        rule_id: "rule_high_error",
        rule_name: "High Engine Error Rate",
        engine_type: "aegis",
        severity: "critical",
        state: "firing",
        value: 0.072,
        threshold: 0.05,
        triggered_at: "2026-09-03T10:05:00Z",
      },
    ];
    expect(alertEventListSchema.safeParse(events).success).toBe(true);
  });

  it("validates sloListSchema", () => {
    const slos = [
      {
        id: "slo_nexus_avail",
        name: "Nexus Availability",
        engine_type: "nexus",
        target_percentage: 99.9,
        current_percentage: 99.98,
        error_budget_remaining_percent: 82.5,
        burn_rate: 0.65,
        window: "30d",
      },
    ];
    expect(sloListSchema.safeParse(slos).success).toBe(true);
  });
});
