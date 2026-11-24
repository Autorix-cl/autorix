import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { NextRequest, NextResponse } from "next/server";
import { requireOperator } from "@/lib/auth/session";
import type { Operator } from "@/lib/auth/types";
import { GET as getSubject } from "./subject/[id]/route";
import { POST as evaluateEffectiveAccess } from "./effective-access/route";
import { safePatternMatch } from "@/lib/explorer/pattern-match";
import { POST as simulateRequest } from "./simulate/route";
import { GET as checkConsistency } from "./consistency/route";

vi.mock("@/lib/auth/session", () => ({
  requireOperator: vi.fn(),
  getCurrentOperator: vi.fn(),
  SESSION_COOKIE_NAME: "autorix_session",
}));

describe("Cross-Engine Explorer BFF Routes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(requireOperator).mockResolvedValue({
      ok: true,
      operator: {
        id: "op-1",
        email: "admin@autorix.io",
        name: "Admin",
        role: "admin",
        is_local: true,
        is_active: true,
      } as Operator,
    });
    global.fetch = vi.fn();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("rejects unauthenticated requests across all explorer endpoints", async () => {
    vi.mocked(requireOperator).mockResolvedValue({
      ok: false,
      response: NextResponse.json({ error: "unauthorized: authentication required" }, { status: 401 }),
    });

    expect(
      (
        await getSubject(new NextRequest("http://localhost/api/explorer/subject/alice"), {
          params: Promise.resolve({ id: "alice" }),
        })
      ).status,
    ).toBe(401);
    expect(
      (
        await evaluateEffectiveAccess(
          new NextRequest("http://localhost/api/explorer/effective-access", {
            method: "POST",
            body: JSON.stringify({}),
          }),
        )
      ).status,
    ).toBe(401);
    expect(
      (
        await simulateRequest(
          new NextRequest("http://localhost/api/explorer/simulate", { method: "POST", body: JSON.stringify({}) }),
        )
      ).status,
    ).toBe(401);
    expect((await checkConsistency()).status).toBe(401);
  });

  it("aggregates unified subject across Ego, Nexus, Vulcan, Hermes, and Janus without leaking system keys", async () => {
    vi.mocked(global.fetch).mockImplementation(async (url: RequestInfo | URL) => {
      const u = String(url);
      if (u.includes("/admin/identities/alice/sessions")) {
        return new Response(JSON.stringify([{ id: "sess-1", identity_id: "alice", active: true }]), { status: 200 });
      }
      if (u.includes("/admin/identities/alice")) {
        return new Response(JSON.stringify({ id: "alice", state: "active", traits: { email: "alice@autorix.io" } }), {
          status: 200,
        });
      }
      if (u.includes("/tuples")) {
        return new Response(
          JSON.stringify({
            data: [
              { namespace: "document", object: "doc_1", relation: "owner", subject_id: "alice" },
              { namespace: "document", object: "doc_2", relation: "viewer", subject_id: "alice-admin" },
            ],
          }),
          { status: 200 },
        );
      }
      if (u.includes("/keys")) {
        return new Response(
          JSON.stringify({
            data: [
              { id: "k-1", name: "Alice CLI", key_prefix: "av_live_", owner_id: "alice", call_count: 10 },
              { id: "k-sys", name: "System Key", key_prefix: "av_live_", owner_id: "system", call_count: 99 },
              { id: "k-orphan", name: "Unowned Key", key_prefix: "av_live_", call_count: 5 },
            ],
          }),
          { status: 200 },
        );
      }
      if (u.includes("/scim/v2/Users")) {
        return new Response(
          JSON.stringify({
            Resources: [{ id: "u-1", userName: "alice", externalId: "alice", active: true }],
          }),
          { status: 200 },
        );
      }
      if (u.includes("/admin/grants")) {
        return new Response(
          JSON.stringify([{ id: "gr-1", client_id: "console-app", scope: "openid email", subject: "alice" }]),
          { status: 200 },
        );
      }
      return new Response(JSON.stringify({}), { status: 200 });
    });

    const res = await getSubject(new NextRequest("http://localhost/api/explorer/subject/alice"), {
      params: Promise.resolve({ id: "alice" }),
    });
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.id).toBe("alice");
    expect(json.identity?.traits?.email).toBe("alice@autorix.io");
    expect(json.sessions).toHaveLength(1);
    expect(json.relations).toHaveLength(1);
    expect(json.api_keys).toHaveLength(1);
    expect(json.api_keys[0].id).toBe("k-1");
    expect(json.oauth_grants).toHaveLength(1);
    expect(json.enterprise_linkage?.external_id).toBe("alice");
  });

  it("returns 404 when Ego reports identity not found instead of fabricating a fake email", async () => {
    vi.mocked(global.fetch).mockImplementation(async (url: RequestInfo | URL) => {
      if (String(url).includes("/admin/identities/ghost")) {
        return new Response(JSON.stringify({ error: "not found" }), { status: 404 });
      }
      return new Response(JSON.stringify([]), { status: 200 });
    });

    const res = await getSubject(new NextRequest("http://localhost/api/explorer/subject/ghost"), {
      params: Promise.resolve({ id: "ghost" }),
    });
    expect(res.status).toBe(404);
  });

  it("evaluates multi-engine effective access when all engines allow and fails closed on unreachable engines", async () => {
    vi.mocked(global.fetch).mockImplementation(async (url: RequestInfo | URL) => {
      const u = String(url);
      if (u.includes("/rules")) {
        return new Response(
          JSON.stringify([
            { id: "r1", name: "docs-rule", enabled: true, path: "/api/v1/documents", upstream_url: "http://docs:8080" },
          ]),
          { status: 200 },
        );
      }
      if (u.includes("/check")) {
        return new Response(JSON.stringify({ allowed: true }), { status: 200 });
      }
      if (u.includes("/policies/evaluate")) {
        return new Response(JSON.stringify({ all_passed: true, total_evaluated: 2, results: [] }), { status: 200 });
      }
      return new Response(JSON.stringify({}), { status: 404 });
    });

    const res = await evaluateEffectiveAccess(
      new NextRequest("http://localhost/api/explorer/effective-access", {
        method: "POST",
        body: JSON.stringify({
          subject: "user:alice",
          resource: "/api/v1/documents/doc_1",
          action: "GET",
        }),
      }),
    );

    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.subject).toBe("user:alice");
    expect(json.allowed).toBe(true);
    expect(json.engine_breakdown.aegis.matched).toBe(true);
    expect(json.engine_breakdown.nexus.allowed).toBe(true);
    expect(json.engine_breakdown.themis.allowed).toBe(true);

    // Now simulate engine outages: MUST fail closed!
    vi.mocked(global.fetch).mockRejectedValue(new TypeError("connect ECONNREFUSED"));
    const failRes = await evaluateEffectiveAccess(
      new NextRequest("http://localhost/api/explorer/effective-access", {
        method: "POST",
        body: JSON.stringify({ subject: "user:alice", resource: "/api/v1/documents/doc_1", action: "GET" }),
      }),
    );
    expect(failRes.status).toBe(200);
    const failJson = await failRes.json();
    expect(failJson.allowed).toBe(false);
    expect(failJson.engine_breakdown.aegis.matched).toBe(false);
    expect(failJson.engine_breakdown.nexus.allowed).toBe(false);
    expect(failJson.engine_breakdown.themis.allowed).toBe(false);

    // ReDoS protection guard
    expect(safePatternMatch("(a+)+$", "aaaaaaaaaaaaaaaaaaaaaa!")).toBe(false);
    expect(safePatternMatch("a".repeat(300), "a")).toBe(false);
  });

  it("simulates end-to-end request trace against real Aegis, Nexus, and Themis endpoints", async () => {
    vi.mocked(global.fetch).mockImplementation(async (url: RequestInfo | URL) => {
      const u = String(url);
      if (u.includes("/rules/test-match")) {
        return new Response(JSON.stringify({ matched: true, rule_id: "r-projects", final_verdict: "allow" }), {
          status: 200,
        });
      }
      if (u.includes("/check")) {
        return new Response(JSON.stringify({ allowed: true, reason: "direct tuple" }), { status: 200 });
      }
      if (u.includes("/policies/evaluate")) {
        return new Response(JSON.stringify({ all_passed: true, total_evaluated: 1 }), { status: 200 });
      }
      return new Response(JSON.stringify({}), { status: 404 });
    });

    const res = await simulateRequest(
      new NextRequest("http://localhost/api/explorer/simulate", {
        method: "POST",
        body: JSON.stringify({
          method: "GET",
          path: "/api/v1/projects/alpha",
          headers: { authorization: "Bearer av_live_test_key" },
          subject: "user:bob",
        }),
      }),
    );

    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.trace_id).toBeDefined();
    expect(json.steps.length).toBeGreaterThanOrEqual(4);
    expect(json.outcome.allowed).toBe(true);
    expect(json.outcome.status_code).toBe(200);

    // When engines are unreachable, simulate fails closed with 503 outcome
    vi.mocked(global.fetch).mockRejectedValue(new TypeError("offline"));
    const failSim = await simulateRequest(
      new NextRequest("http://localhost/api/explorer/simulate", {
        method: "POST",
        body: JSON.stringify({ method: "GET", path: "/api/v1/projects/alpha", subject: "user:bob" }),
      }),
    );
    const failData = await failSim.json();
    expect(failData.outcome.allowed).toBe(false);
    expect(failData.outcome.status_code).toBe(503);
  });

  it("runs cross-engine configuration consistency checks using Nexus /admin/namespaces and flags drift", async () => {
    const fetchMock = vi.fn().mockImplementation(async (url: RequestInfo | URL) => {
      const u = String(url);
      if (u.includes("/rules")) {
        return new Response(
          JSON.stringify([
            {
              id: "r1",
              name: "billing",
              enabled: true,
              upstream_url: "http://billing:8080",
              authorizer: { config: { namespace: "missing_ns" } },
            },
          ]),
          { status: 200 },
        );
      }
      if (u.includes("/admin/namespaces")) {
        return new Response(JSON.stringify([{ name: "document" }]), { status: 200 });
      }
      return new Response(JSON.stringify([]), { status: 200 });
    });
    vi.stubGlobal("fetch", fetchMock);

    const res = await checkConsistency();
    expect(res.status).toBe(200);
    const findings = await res.json();
    expect(Array.isArray(findings)).toBe(true);
    expect(fetchMock.mock.calls.some(([u]) => String(u).includes("/admin/namespaces"))).toBe(true);
    expect(fetchMock.mock.calls.some(([u]) => String(u).includes("/schema"))).toBe(false);
    expect(findings.some((f: { id: string }) => f.id === "aegis-nexus-orphaned-namespaces")).toBe(true);
    expect(findings.some((f: { id: string }) => f.id === "aegis-nexus-alignment")).toBe(false);
  });
});
