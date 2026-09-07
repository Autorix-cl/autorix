import { describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { PerEngineDashboard } from "./per-engine-dashboard";
import { LogViewer } from "./log-viewer";
import { TraceExplorer } from "./trace-explorer";
import { AlertsManager } from "./alerts-manager";
import { SLODashboard } from "./slo-dashboard";

describe("truthful observability UI", () => {
  it("does not render fabricated per-engine statistics", () => { render(<PerEngineDashboard initialEngine="nexus" />); expect(screen.getByText(/recording rules are not configured/i)).toBeInTheDocument(); expect(screen.queryByText(/4.8 ms/i)).not.toBeInTheDocument(); });
  it("identifies unavailable log and trace sources", () => { render(<><LogViewer /><TraceExplorer /></>); expect(screen.getByText(/No log aggregation backend/i)).toBeInTheDocument(); expect(screen.getByText(/No trace backend/i)).toBeInTheDocument(); });
  it("identifies SLO source requirements", () => { render(<SLODashboard />); expect(screen.getByText(/SLO recording rules are not configured/i)).toBeInTheDocument(); });
  it("makes Prometheus alerts read-only", async () => { vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify([]), { status: 200 })))); render(<AlertsManager />); await waitFor(() => expect(screen.getByText(/managed by Alertmanager/i)).toBeInTheDocument()); expect(screen.queryByText("Ack")).not.toBeInTheDocument(); expect(screen.queryByText("Silence")).not.toBeInTheDocument(); });
});
