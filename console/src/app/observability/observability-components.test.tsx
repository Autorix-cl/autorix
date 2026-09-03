import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import * as React from "react";
import { FleetDashboard } from "./fleet-dashboard";
import { PerEngineDashboard } from "./per-engine-dashboard";
import { LogViewer } from "./log-viewer";
import { TraceExplorer } from "./trace-explorer";
import { AlertsManager } from "./alerts-manager";
import { SLODashboard } from "./slo-dashboard";
import ObservabilityPage from "./page";

describe("Observability UI Components", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes("/metrics")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              timestamp: "2026-09-03T10:00:00Z",
              total_engines: 7,
              total_instances: 14,
              healthy_instances: 14,
              requests_total: 248900,
              fleet_qps: 142.8,
              fleet_error_rate: 0.006,
              fleet_latency_p95_ms: 3.8,
              auth_decisions_total: 124500,
              auth_allow_rate: 0.988,
              engines: [
                {
                  engine_type: "nexus",
                  status: "healthy",
                  instance_count: 2,
                  requests_total: 46000,
                  requests_per_second: 32.1,
                  error_rate: 0.005,
                  latency_p50_ms: 1.9,
                  latency_p95_ms: 4.8,
                  latency_p99_ms: 11.2,
                  auth_decisions_total: 46000,
                  auth_allow_rate: 0.985,
                },
              ],
            }),
        });
      }
      if (url.includes("/logs")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve([
              {
                id: "log-1",
                timestamp: "2026-09-03T10:00:00Z",
                engine: "aegis",
                level: "info",
                message: "Proxy request routed",
                request_id: "req-1",
                attributes: {},
              },
            ]),
        });
      }
      if (url.includes("/traces")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve([
              {
                trace_id: "tr-test",
                root_operation: "HTTP GET /test",
                root_engine: "aegis",
                total_duration_ms: 4.2,
                timestamp: "2026-09-03T10:00:00Z",
                status: "ok",
                spans: [
                  {
                    span_id: "sp-1",
                    trace_id: "tr-test",
                    engine: "aegis",
                    operation: "route",
                    status: "ok",
                    start_time_ms: 0,
                    duration_ms: 4.2,
                    attributes: {},
                  },
                ],
              },
            ]),
        });
      }
      if (url.includes("/alerts/rules")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve([
              {
                id: "rule-1",
                name: "High Ingress Error Rate",
                engine_type: "aegis",
                severity: "warning",
                metric: "errors",
                threshold: 0.02,
                operator: "gt",
                duration: "2m",
                enabled: true,
              },
            ]),
        });
      }
      if (url.includes("/alerts")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve([
              {
                id: "evt-1",
                rule_id: "rule-1",
                rule_name: "High Ingress Error Rate",
                engine_type: "aegis",
                severity: "warning",
                state: "firing",
                value: 0.03,
                threshold: 0.02,
                triggered_at: "2026-09-03T10:00:00Z",
              },
            ]),
        });
      }
      if (url.includes("/slo")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve([
              {
                id: "slo-1",
                name: "Nexus Availability",
                engine_type: "nexus",
                target_percentage: 99.9,
                current_percentage: 99.95,
                error_budget_remaining_percent: 75.0,
                burn_rate: 0.5,
                window: "30d",
              },
            ]),
        });
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) });
    });
  });

  it("renders FleetDashboard with metric values", async () => {
    render(<FleetDashboard />);
    await waitFor(() => {
      expect(screen.getByText(/Fleet Telemetry & RED Overview/i)).toBeInTheDocument();
      expect(screen.getByText(/Fleet Throughput/i)).toBeInTheDocument();
    });
  });

  it("renders PerEngineDashboard with domain specific metrics", () => {
    render(<PerEngineDashboard initialEngine="nexus" />);
    expect(screen.getByText(/Zanzibar Check Latency/i)).toBeInTheDocument();
    expect(screen.getByText(/Graph Traversal Depth/i)).toBeInTheDocument();
  });

  it("renders LogViewer and displays logs", async () => {
    render(<LogViewer />);
    await waitFor(() => {
      expect(screen.getByText(/Structured Fleet Logs/i)).toBeInTheDocument();
      expect(screen.getByText(/Proxy request routed/i)).toBeInTheDocument();
    });
  });

  it("renders TraceExplorer with trace details", async () => {
    render(<TraceExplorer />);
    await waitFor(() => {
      expect(screen.getByText(/Distributed Trace Waterfall/i)).toBeInTheDocument();
      expect(screen.getAllByText(/tr-test/i).length).toBeGreaterThanOrEqual(1);
    });
  });

  it("renders AlertsManager and SLODashboard", async () => {
    render(<AlertsManager />);
    await waitFor(() => {
      expect(screen.getByText(/High Ingress Error Rate/i)).toBeInTheDocument();
    });

    render(<SLODashboard />);
    await waitFor(() => {
      expect(screen.getByText(/Nexus Availability/i)).toBeInTheDocument();
    });
  });

  it("renders ObservabilityPage container with tabs", () => {
    render(<ObservabilityPage />);
    expect(screen.getByText(/Fleet Observability & Telemetry/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Fleet Overview/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Engine Deep Dive/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Structured Logs/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Trace Explorer/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Alerts & Rules/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /SLOs & Budgets/i })).toBeInTheDocument();
  });
});
