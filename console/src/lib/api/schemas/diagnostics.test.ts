import { describe, it, expect } from "vitest";
import {
  connectivityProbeResultSchema,
  configDriftListSchema,
  migrationStatusListSchema,
  changeTimelineListSchema,
  diagnosticBundleSchema,
} from "./diagnostics";

describe("Diagnostics Zod Schemas", () => {
  it("validates connectivityProbeResultSchema", () => {
    const probe = {
      probe_id: "prb_100",
      timestamp: "2026-09-03T10:00:00Z",
      source_instance_id: "aegis-1",
      source_engine: "aegis",
      target_instance_id: "nexus-1",
      target_engine: "nexus",
      target_endpoint: "http://nexus:8080/health/ready",
      dns_status: "healthy",
      dns_latency_ms: 0.8,
      tcp_status: "healthy",
      tcp_latency_ms: 1.2,
      tls_status: "skipped",
      tls_latency_ms: 0,
      http_status: "healthy",
      http_code: 200,
      http_latency_ms: 2.4,
      overall_status: "healthy",
    };
    expect(connectivityProbeResultSchema.safeParse(probe).success).toBe(true);
  });

  it("validates configDriftListSchema", () => {
    const findings = [
      {
        id: "drift_1",
        engine_type: "aegis",
        environment: "production",
        parameter: "proxy_timeout_seconds",
        reference_value: "30",
        divergent_instances: [{ instance_id: "aegis-2", value: "15" }],
        severity: "warning",
      },
    ];
    expect(configDriftListSchema.safeParse(findings).success).toBe(true);
  });

  it("validates migrationStatusListSchema", () => {
    const migrations = [
      {
        engine_type: "ego",
        instance_id: "ego-1",
        environment: "production",
        current_schema_version: 5,
        expected_schema_version: 5,
        up_to_date: true,
        pending_migrations_count: 0,
        last_migrated_at: "2026-09-02T12:00:00Z",
      },
    ];
    expect(migrationStatusListSchema.safeParse(migrations).success).toBe(true);
  });

  it("validates changeTimelineListSchema", () => {
    const events = [
      {
        id: "ev_1",
        timestamp: "2026-09-03T09:45:00Z",
        event_type: "config_change",
        engine_type: "aegis",
        title: "Proxy Rule Route Modified",
        description: "Updated upstream timeout from 15s to 30s",
        actor: "operator:alice",
        correlated_error_spike: false,
      },
    ];
    expect(changeTimelineListSchema.safeParse(events).success).toBe(true);
  });

  it("validates diagnosticBundleSchema", () => {
    const bundle = {
      bundle_id: "diag_20260903_1100",
      created_at: "2026-09-03T11:00:00Z",
      fleet_summary: {
        total_engines: 7,
        healthy_engines: 7,
        total_instances: 14,
      },
      engines_included: ["aegis", "nexus", "themis"],
      events_count: 50,
      logs_count: 200,
      redacted: true,
      download_url: "/api/diagnostics/bundle/diag_20260903_1100.tar.gz",
    };
    expect(diagnosticBundleSchema.safeParse(bundle).success).toBe(true);
  });
});
