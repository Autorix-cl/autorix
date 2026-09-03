import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import * as React from "react";
import { UnifiedSubjectView } from "./unified-subject-view";
import { EffectiveAccessExplorer } from "./effective-access-explorer";
import { RequestSimulator } from "./request-simulator";
import { ConsistencyChecker } from "./consistency-checker";
import ExplorerPage from "./page";

describe("Cross-Engine Explorer UI Components", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn();
  });

  it("renders UnifiedSubjectView and displays resolved identity and relations", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: "alice",
          identity: { id: "alice", state: "active", traits: { email: "alice@autorix.io" } },
          sessions: [{ id: "s1", ip_address: "192.168.1.1", user_agent: "Firefox" }],
          relations: [{ namespace: "document", object: "doc_1", relation: "owner", subject: "user:alice" }],
          oauth_grants: [],
          api_keys: [{ id: "k1", name: "Deploy Key", prefix: "av_live_****" }],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    render(<UnifiedSubjectView />);

    await waitFor(() => {
      expect(screen.getByText("Ego Identity Profile")).toBeDefined();
      expect(screen.getByText("alice@autorix.io")).toBeDefined();
      expect(screen.getByText("document:doc_1")).toBeDefined();
    });
  });

  it("renders EffectiveAccessExplorer and renders engine breakdown", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          subject: "user:alice",
          resource: "/api/v1/finance/invoices",
          action: "GET",
          allowed: true,
          reason: "Permitted across engines",
          engine_breakdown: {
            aegis: { matched: true, rule_name: "finance-rule", upstream: "http://upstream:8080" },
            nexus: { checked: true, namespace: "finance", object: "invoices", relation: "viewer", allowed: true },
            themis: { evaluated: true, policies_matched: 1, allowed: true },
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    render(<EffectiveAccessExplorer />);

    await waitFor(() => {
      expect(screen.getByText("ACCESS GRANTED · 200 OK")).toBeDefined();
      expect(screen.getByText("Aegis Ingress")).toBeDefined();
      expect(screen.getByText("Nexus ReBAC")).toBeDefined();
      expect(screen.getByText("Themis ABAC")).toBeDefined();
    });
  });

  it("renders RequestSimulator and runs simulation", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          trace_id: "sim_test_1",
          request: { method: "GET", path: "/api/test", headers: {}, subject: "user:bob" },
          steps: [
            { step: "Aegis Route Match", engine: "aegis", status: "pass", latency_ms: 0.5, details: {} },
            { step: "Nexus ReBAC Check", engine: "nexus", status: "pass", latency_ms: 1.0, details: {} },
          ],
          outcome: { allowed: true, status_code: 200, final_decision: "Forwarded", latency_total_ms: 1.5 },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    render(<RequestSimulator />);

    await waitFor(() => {
      expect(screen.getByText("End-to-End Request Pipeline Simulator")).toBeDefined();
      expect(screen.getByText("Aegis Route Match")).toBeDefined();
      expect(screen.getByText("Nexus ReBAC Check")).toBeDefined();
    });
  });

  it("renders ConsistencyChecker and displays findings", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify([
          {
            id: "f1",
            severity: "warning",
            category: "vulcan",
            title: "Stale API Keys Detected",
            description: "Some keys have zero traffic in 30 days.",
            remediation: "Revoke stale keys",
            remediation_link: "/vulcan",
          },
        ]),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    render(<ConsistencyChecker />);

    await waitFor(() => {
      expect(screen.getByText("Stale API Keys Detected")).toBeDefined();
      expect(screen.getByText("1 Warnings")).toBeDefined();
    });
  });

  it("renders ExplorerPage with tabs", async () => {
    vi.mocked(global.fetch).mockResolvedValue(
      new Response(JSON.stringify({}), { status: 200, headers: { "Content-Type": "application/json" } })
    );

    render(<ExplorerPage />);

    expect(screen.getByText("Cross-Engine Intelligence & Explorer")).toBeDefined();
    expect(screen.getByText("Unified Subject")).toBeDefined();
    expect(screen.getByText("Effective Access")).toBeDefined();
    expect(screen.getByText("Request Simulator")).toBeDefined();
    expect(screen.getByText("Consistency Checks")).toBeDefined();
  });
});
