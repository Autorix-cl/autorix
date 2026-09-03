import { describe, it, expect, vi, beforeEach } from "vitest";
import { NextRequest } from "next/server";
import { GET as getSubject } from "./subject/[id]/route";
import { POST as evaluateEffectiveAccess } from "./effective-access/route";
import { POST as simulateRequest } from "./simulate/route";
import { GET as checkConsistency } from "./consistency/route";

describe("Cross-Engine Explorer BFF Routes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn();
  });

  it("aggregates unified subject across Ego, Nexus, Vulcan, and Hermes", async () => {
    // 1. Mock Ego identity
    vi.mocked(global.fetch).mockImplementation(async (url: RequestInfo | URL) => {
      const u = String(url);
      if (u.includes("/identities/alice/sessions")) {
        return new Response(JSON.stringify([{ id: "sess-1", identity_id: "alice", active: true }]), { status: 200 });
      }
      if (u.includes("/identities/alice")) {
        return new Response(
          JSON.stringify({ id: "alice", state: "active", traits: { email: "alice@autorix.io" } }),
          { status: 200 }
        );
      }
      if (u.includes("/tuples")) {
        return new Response(
          JSON.stringify({
            data: [{ namespace: "document", object: "doc_1", relation: "owner", subject_id: "alice" }],
          }),
          { status: 200 }
        );
      }
      if (u.includes("/keys")) {
        return new Response(
          JSON.stringify({
            data: [{ id: "k-1", name: "Alice CLI", key_prefix: "av_live_", owner_id: "alice", call_count: 10 }],
          }),
          { status: 200 }
        );
      }
      if (u.includes("/scim/v2/Users")) {
        return new Response(
          JSON.stringify({
            Resources: [{ id: "u-1", userName: "alice", externalId: "alice", active: true }],
          }),
          { status: 200 }
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
    expect(json.enterprise_linkage?.external_id).toBe("alice");
  });

  it("evaluates multi-engine effective access", async () => {
    const res = await evaluateEffectiveAccess(
      new NextRequest("http://localhost/api/explorer/effective-access", {
        method: "POST",
        body: JSON.stringify({
          subject: "user:alice",
          resource: "/api/v1/documents/doc_1",
          action: "GET",
        }),
      })
    );

    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.subject).toBe("user:alice");
    expect(json.allowed).toBe(true);
    expect(json.engine_breakdown.aegis.matched).toBe(true);
    expect(json.engine_breakdown.nexus.checked).toBe(true);
    expect(json.engine_breakdown.themis.evaluated).toBe(true);
  });

  it("simulates end-to-end request trace", async () => {
    const res = await simulateRequest(
      new NextRequest("http://localhost/api/explorer/simulate", {
        method: "POST",
        body: JSON.stringify({
          method: "GET",
          path: "/api/v1/projects/alpha",
          headers: { authorization: "Bearer av_live_test_key" },
          subject: "user:bob",
        }),
      })
    );

    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.trace_id).toBeDefined();
    expect(json.steps.length).toBeGreaterThanOrEqual(4);
    expect(json.outcome.allowed).toBe(true);
  });

  it("runs cross-engine configuration consistency checks", async () => {
    const res = await checkConsistency();
    expect(res.status).toBe(200);
    const findings = await res.json();
    expect(Array.isArray(findings)).toBe(true);
    expect(findings.length).toBeGreaterThanOrEqual(1);
  });
});
