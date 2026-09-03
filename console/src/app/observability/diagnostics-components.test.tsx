import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import * as React from "react";
import { DiagnosticsManager } from "./diagnostics-manager";

function mockResponse(data: unknown) {
  return Promise.resolve({
    ok: true,
    status: 200,
    headers: new Headers(),
    text: () => Promise.resolve(JSON.stringify(data)),
    json: () => Promise.resolve(data),
  } as unknown as Response);
}

describe("Diagnostics UI Components", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (url.includes("/probe") && opts?.method === "POST") {
        return mockResponse({
          probe_id: "prb_123",
          source_engine: "aegis",
          target_engine: "nexus",
          target_endpoint: "http://nexus:8080/health/ready",
          dns_status: "healthy",
          dns_latency_ms: 0.6,
          tcp_status: "healthy",
          tcp_latency_ms: 1.1,
          tls_status: "skipped",
          tls_latency_ms: 0,
          http_status: "healthy",
          http_code: 200,
          http_latency_ms: 2.2,
          overall_status: "healthy",
        });
      }
      if (url.includes("/drift")) {
        return mockResponse([
          {
            id: "drift-1",
            engine_type: "aegis",
            environment: "production",
            parameter: "proxy.upstream_timeout_ms",
            reference_value: "5000",
            divergent_instances: [{ instance_id: "aegis-2", value: "3000" }],
            severity: "warning",
          },
        ]);
      }
      if (url.includes("/migrations")) {
        return mockResponse([
          {
            engine_type: "nexus",
            instance_id: "nexus-1",
            environment: "production",
            current_schema_version: 4,
            expected_schema_version: 4,
            up_to_date: true,
            pending_migrations_count: 0,
            last_migrated_at: "2026-09-02T11:45:00Z",
          },
        ]);
      }
      if (url.includes("/timeline")) {
        return mockResponse([
          {
            id: "evt-1",
            timestamp: "2026-09-03T10:00:00Z",
            event_type: "config_change",
            engine_type: "aegis",
            title: "Aegis Ingress Rate Limit Reduced",
            description: "Operator adjusted burst quota",
            actor: "operator:sec_lead",
            correlated_error_spike: true,
            error_spike_rate: 0.038,
          },
        ]);
      }
      if (url.includes("/bundle") && opts?.method === "POST") {
        return mockResponse({
          bundle_id: "diag_20260903_1200",
          created_at: "2026-09-03T12:00:00Z",
          fleet_summary: { total_engines: 7, healthy_engines: 7, total_instances: 14 },
          engines_included: ["aegis", "nexus"],
          events_count: 50,
          logs_count: 200,
          redacted: true,
          download_url: "/api/diagnostics/bundle/diag_20260903_1200.tar.gz",
        });
      }
      return mockResponse({});
    });
  });

  it("renders DiagnosticsManager and executes connectivity probe", async () => {
    render(<DiagnosticsManager />);
    expect(screen.getByText(/Diagnostics & Incident Support/i)).toBeInTheDocument();
    expect(screen.getByText(/Instance-to-Instance Live Probe/i)).toBeInTheDocument();

    const runButton = screen.getByRole("button", { name: /Execute Probe/i });
    fireEvent.click(runButton);

    await waitFor(() => {
      expect(screen.getByText(/1\. DNS Lookup/i)).toBeInTheDocument();
      expect(screen.getByText(/2\. TCP Connect/i)).toBeInTheDocument();
      expect(screen.getByText(/Application HTTP/i)).toBeInTheDocument();
    });
  });

  it("switches to Config Drift tab and displays findings", async () => {
    render(<DiagnosticsManager />);
    const driftTabButton = screen.getByRole("button", { name: /Config Drift/i });
    fireEvent.click(driftTabButton);

    await waitFor(() => {
      expect(screen.getByText(/Divergent Configuration Across Replicas/i)).toBeInTheDocument();
      expect(screen.getByText(/proxy\.upstream_timeout_ms/i)).toBeInTheDocument();
    });
  });

  it("switches to Migration Status tab and displays engine schema versions", async () => {
    render(<DiagnosticsManager />);
    const migTabButton = screen.getByRole("button", { name: /Migration Status/i });
    fireEvent.click(migTabButton);

    await waitFor(() => {
      expect(screen.getByText(/Engine Schema Migration State/i)).toBeInTheDocument();
      expect(screen.getByText(/Current Version/i)).toBeInTheDocument();
    });
  });

  it("switches to Change Correlation Timeline and displays incident events", async () => {
    render(<DiagnosticsManager />);
    const timeTabButton = screen.getByRole("button", { name: /Change Correlation Timeline/i });
    fireEvent.click(timeTabButton);

    await waitFor(() => {
      expect(screen.getByText(/Change Correlation & Incident Overlay/i)).toBeInTheDocument();
      expect(screen.getByText(/Aegis Ingress Rate Limit Reduced/i)).toBeInTheDocument();
      expect(screen.getByText(/Error Spike/i)).toBeInTheDocument();
    });
  });

  it("triggers diagnostic bundle export", async () => {
    render(<DiagnosticsManager />);
    const exportButton = screen.getByRole("button", { name: /Export Diagnostic Bundle/i });
    fireEvent.click(exportButton);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "/api/diagnostics/bundle",
        expect.objectContaining({ method: "POST" })
      );
    });
  });
});
