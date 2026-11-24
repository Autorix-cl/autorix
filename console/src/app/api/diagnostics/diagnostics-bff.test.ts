import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { NextRequest, NextResponse } from "next/server";
import { requireOperator } from "@/lib/auth/session";
import type { Operator } from "@/lib/auth/types";
import { POST as runProbe } from "./probe/route";
import { GET as getDrift } from "./drift/route";
import { GET as getMigrations } from "./migrations/route";
import { GET as getTimeline } from "./timeline/route";
import { GET as downloadBundle, POST as exportBundle } from "./bundle/route";

vi.mock("@/lib/auth/session", () => ({
  requireOperator: vi.fn(),
  getCurrentOperator: vi.fn(),
  SESSION_COOKIE_NAME: "autorix_session",
}));

describe("Diagnostics BFF Routes", () => {
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
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("rejects unauthenticated requests across all diagnostics endpoints", async () => {
    vi.mocked(requireOperator).mockResolvedValue({
      ok: false,
      response: NextResponse.json({ error: "unauthorized: authentication required" }, { status: 401 }),
    });

    expect(
      (await runProbe(new NextRequest("http://localhost/api/diagnostics/probe", { method: "POST", body: JSON.stringify({}) }))).status,
    ).toBe(401);
    expect((await getDrift()).status).toBe(401);
    expect((await getMigrations()).status).toBe(401);
    expect((await getTimeline()).status).toBe(401);
    expect((await exportBundle(new NextRequest("http://localhost/api/diagnostics/bundle", { method: "POST" }))).status).toBe(401);
  });

  it("POST /api/diagnostics/probe performs real timed fetch against target /health/ready and rejects unregistered engines", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(new Response("OK", { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    const req = new NextRequest("http://localhost/api/diagnostics/probe", {
      method: "POST",
      body: JSON.stringify({ source_engine: "aegis", target_engine: "nexus" }),
    });
    const res = await runProbe(req);
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data.source_engine).toBe("aegis");
    expect(data.target_engine).toBe("nexus");
    expect(data.http_code).toBe(200);
    expect(data.overall_status).toBe("healthy");
    expect(fetchMock).toHaveBeenCalledWith(expect.stringMatching(/\/health\/ready$/), expect.any(Object));

    // Reject unregistered engine (SSRF prevention)
    const badRes = await runProbe(
      new NextRequest("http://localhost/api/diagnostics/probe", {
        method: "POST",
        body: JSON.stringify({ target_engine: "http://169.254.169.254" }),
      }),
    );
    expect(badRes.status).toBe(400);

    // Report unreachable engine honestly
    vi.stubGlobal("fetch", vi.fn().mockRejectedValueOnce(new TypeError("connect ECONNREFUSED")));
    const failRes = await runProbe(
      new NextRequest("http://localhost/api/diagnostics/probe", {
        method: "POST",
        body: JSON.stringify({ source_engine: "aegis", target_engine: "nexus" }),
      }),
    );
    const failData = await failRes.json();
    expect(failData.overall_status).toBe("failed");
    expect(failData.error_message).toContain("ECONNREFUSED");
  });

  it("GET /api/diagnostics/drift derives real findings from Argus live instances and labels unavailable on outage", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            data: [
              { id: "aegis-1", engine_type: "aegis", environment: "production", version: "1.2.0" },
              { id: "aegis-2", engine_type: "aegis", environment: "production", version: "1.1.0" },
            ],
          }),
          { status: 200 },
        ),
      ),
    );

    const res = await getDrift();
    expect(res.status).toBe(200);
    expect(res.headers.get("X-Autorix-Source")).toBe("live");
    const data = await res.json();
    expect(Array.isArray(data)).toBe(true);
    expect(data).toHaveLength(1);
    expect(data[0].parameter).toBe("engine.version");

    vi.stubGlobal("fetch", vi.fn().mockRejectedValueOnce(new Error("argus offline")));
    const offlineRes = await getDrift();
    expect(offlineRes.status).toBe(503);
    expect((await offlineRes.json()).source).toBe("unavailable");
  });

  it("GET /api/diagnostics/migrations reports live instance migration state and unavailable on outage", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            data: [
              {
                id: "ego-1",
                engine_type: "ego",
                environment: "production",
                schema_version: 6,
                expected_schema_version: 6,
                updated_at: "2026-09-02T10:15:00Z",
              },
            ],
          }),
          { status: 200 },
        ),
      ),
    );

    const res = await getMigrations();
    expect(res.status).toBe(200);
    expect(res.headers.get("X-Autorix-Source")).toBe("live");
    const data = await res.json();
    expect(Array.isArray(data)).toBe(true);
    expect(data).toHaveLength(1);
    expect(data[0].up_to_date).toBe(true);

    vi.stubGlobal("fetch", vi.fn().mockRejectedValueOnce(new Error("argus offline")));
    const offlineRes = await getMigrations();
    expect(offlineRes.status).toBe(503);
    expect((await offlineRes.json()).source).toBe("unavailable");
  });

  it("GET /api/diagnostics/timeline maps live Argus audit events and returns unavailable on outage", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(async (url: RequestInfo | URL) => {
        if (String(url).includes("/v1/audit")) {
          return new Response(
            JSON.stringify({
              data: [
                {
                  id: "aud-1",
                  action: "update",
                  resource_type: "proxy_rule",
                  engine_type: "aegis",
                  actor: "admin@autorix.io",
                  outcome: "failure",
                  occurred_at: "2026-09-02T12:00:00Z",
                },
              ],
            }),
            { status: 200 },
          );
        }
        return new Response(JSON.stringify({ data: [] }), { status: 200 });
      }),
    );

    const res = await getTimeline();
    expect(res.status).toBe(200);
    expect(res.headers.get("X-Autorix-Source")).toBe("live");
    const data = await res.json();
    expect(Array.isArray(data)).toBe(true);
    expect(data).toHaveLength(1);
    expect(data[0].correlated_error_spike).toBe(true);

    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("argus offline")));
    const offlineRes = await getTimeline();
    expect(offlineRes.status).toBe(503);
    expect((await offlineRes.json()).source).toBe("unavailable");
  });

  it("POST/GET /api/diagnostics/bundle collects live redacted diagnostics and supports JSON attachment download", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(async (url: RequestInfo | URL) => {
        if (String(url).includes("/health/ready")) {
          return new Response("OK", { status: 200 });
        }
        return new Response(JSON.stringify({ data: [] }), { status: 200 });
      }),
    );

    const res = await exportBundle(new NextRequest("http://localhost/api/diagnostics/bundle", { method: "POST" }));
    expect(res.status).toBe(201);
    const data = await res.json();
    expect(data.bundle_id).toBeTruthy();
    expect(data.redacted).toBe(true);
    expect(data.bundle.probes).toHaveLength(7);
    expect(data.download_url).toBe("/api/diagnostics/bundle?download=1");

    const dlRes = await downloadBundle(new NextRequest("http://localhost/api/diagnostics/bundle?download=1"));
    expect(dlRes.status).toBe(200);
    expect(dlRes.headers.get("Content-Disposition")).toBe('attachment; filename="autorix-diagnostic-bundle.json"');
  });
});
