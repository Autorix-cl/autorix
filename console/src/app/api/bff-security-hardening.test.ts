import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { NextRequest, NextResponse } from "next/server";
import { requireOperator, getCurrentOperator } from "@/lib/auth/session";
import type { Operator } from "@/lib/auth/types";
import { GET as exportAudit } from "./audit/export/route";
import { GET as searchAll } from "./search/route";
import { GET as getNotifications } from "./notifications/route";
import { GET as getEnterpriseMetadata } from "./enterprise/metadata/route";
import { GET as listPolicies } from "./policies/route";
import { getJanusDiscovery, registerConsoleOAuthClient } from "@/lib/auth/sso";

vi.mock("@/lib/auth/session", () => ({
  requireOperator: vi.fn(),
  getCurrentOperator: vi.fn(),
  SESSION_COOKIE_NAME: "autorix_session",
}));

const adminOp = {
  id: "op-1",
  email: "admin@autorix.io",
  name: "Admin",
  role: "admin",
  is_local: true,
  is_active: true,
} as Operator;

describe("Hardened BFF Direct-Fetch & Integration Routes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(requireOperator).mockResolvedValue({ ok: true, operator: adminOp });
    vi.mocked(getCurrentOperator).mockResolvedValue(adminOp);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("enforces authentication on direct-fetch routes (audit export, search, notifications, enterprise metadata)", async () => {
    vi.mocked(requireOperator).mockResolvedValue({
      ok: false,
      response: NextResponse.json({ error: "unauthorized: authentication required" }, { status: 401 }),
    });

    expect((await exportAudit(new NextRequest("http://localhost/api/audit/export?format=csv"))).status).toBe(401);
    expect((await searchAll(new Request("http://localhost/api/search?q=alice"))).status).toBe(401);
    expect((await getNotifications()).status).toBe(401);
    expect((await getEnterpriseMetadata()).status).toBe(401);
  });

  it("streams raw CSV from Argus /v1/audit/export preserving Content-Type and Content-Disposition", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(
      new Response("id,action,actor\naud_1,CREATE,admin@autorix.io\n", {
        status: 200,
        headers: {
          "Content-Type": "text/csv",
          "Content-Disposition": "attachment; filename=audit-records.csv",
        },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const res = await exportAudit(new NextRequest("http://localhost/api/audit/export?format=csv"));
    expect(res.status).toBe(200);
    expect(res.headers.get("Content-Type")).toContain("text/csv");
    expect(res.headers.get("Content-Disposition")).toContain("audit-records.csv");
    expect(await res.text()).toContain("aud_1,CREATE,admin@autorix.io");
    expect(String(fetchMock.mock.calls[0][0])).toContain("/v1/audit/export?format=csv");
  });

  it("queries Ego /admin/identities and Vulcan /keys in parallel on /api/search", async () => {
    const fetchMock = vi.fn().mockImplementation(async (url: RequestInfo | URL) => {
      const u = String(url);
      if (u.includes("/admin/identities")) {
        return new Response(
          JSON.stringify({ data: [{ id: "id-1", state: "active", traits: { email: "alice@autorix.io" } }] }),
          { status: 200 },
        );
      }
      if (u.includes("/keys")) {
        return new Response(JSON.stringify({ data: [{ id: "k-1", name: "alice-cli-key" }] }), { status: 200 });
      }
      return new Response(JSON.stringify([]), { status: 200 });
    });
    vi.stubGlobal("fetch", fetchMock);

    const res = await searchAll(new Request("http://localhost/api/search?q=alice"));
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data.results.some((r: { type: string }) => r.type === "identity")).toBe(true);
    expect(data.results.some((r: { type: string }) => r.type === "api_key")).toBe(true);
    expect(fetchMock.mock.calls.some(([u]) => String(u).includes("/admin/identities"))).toBe(true);
  });

  it("generates deterministic fallback IDs in /api/notifications instead of Math.random()", async () => {
    const makeResponse = () =>
      new Response(
        JSON.stringify({
          data: [{ action: "mint", engine_type: "ego", actor: "admin", occurred_at: "2026-01-01T00:00:00Z" }],
        }),
        { status: 200 },
      );

    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(async (url: RequestInfo | URL) => {
        if (String(url).includes("/v1/enrollment-audit")) return makeResponse();
        return new Response(JSON.stringify({ data: [] }), { status: 200 });
      }),
    );

    const first = await (await getNotifications()).json();
    const second = await (await getNotifications()).json();
    expect(first.notifications[0].id).toBe(second.notifications[0].id);
    expect(first.notifications[0].id).toMatch(/^audit-[a-f0-9]{12}$/);
  });

  it("returns next_cursor alongside cursor from GET /api/policies", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            data: [{ id: "p1", name: "P1", expression: "true", created_at: "2026-01-01T00:00:00Z" }],
            has_more: true,
            cursor: "cur_123",
          }),
          { status: 200 },
        ),
      ),
    );

    const res = await listPolicies(new NextRequest("http://localhost/api/policies"));
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data.cursor).toBe("cur_123");
    expect(data.next_cursor).toBe("cur_123");
  });

  it("uses /.well-known/openid-configuration for Janus discovery and private admin /admin/clients for registration", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          issuer: "http://janus:4444",
          authorization_endpoint: "http://janus:4444/oauth2/auth",
          token_endpoint: "http://janus:4444/oauth2/token",
          jwks_uri: "http://janus:4444/.well-known/jwks.json",
          response_types_supported: ["code"],
          subject_types_supported: ["public"],
          id_token_signing_alg_values_supported: ["RS256"],
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const discovery = await getJanusDiscovery();
    expect(String(fetchMock.mock.calls[0][0])).toContain("/.well-known/openid-configuration");
    expect(discovery?.authorization_endpoint).toBe("http://localhost:4444/oauth2/auth");

    await registerConsoleOAuthClient();
    const regUrl = String(fetchMock.mock.calls[1][0]);
    expect(regUrl).toContain(":4445/admin/clients");
    expect(regUrl).not.toContain("/oauth2/clients");
  });
});
