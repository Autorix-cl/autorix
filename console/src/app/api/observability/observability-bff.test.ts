import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { NextResponse } from "next/server";
import { requireOperator } from "@/lib/auth/session";
import type { Operator } from "@/lib/auth/types";
import { GET as getMetrics } from "./metrics/route";
import { GET as getTraces } from "./traces/route";
import { GET as getLogs } from "./logs/route";
import { GET as getAlerts } from "./alerts/route";
import { GET as getAlertRules, POST as createAlertRule } from "./alerts/rules/route";

vi.mock("@/lib/auth/session", () => ({
  requireOperator: vi.fn(),
  getCurrentOperator: vi.fn(),
  SESSION_COOKIE_NAME: "autorix_session",
}));

describe("Observability BFF routes", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.mocked(requireOperator).mockResolvedValue({
      ok: true,
      operator: {
        id: "op-1",
        email: "auditor@autorix.io",
        name: "Auditor",
        role: "auditor",
        is_local: true,
        is_active: true,
      } as Operator,
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("rejects unauthenticated requests with 401", async () => {
    vi.mocked(requireOperator).mockResolvedValue({
      ok: false,
      response: NextResponse.json({ error: "unauthorized: authentication required" }, { status: 401 }),
    });
    expect((await getMetrics()).status).toBe(401);
    expect((await getLogs()).status).toBe(401);
    expect((await getTraces()).status).toBe(401);
    expect((await getAlerts()).status).toBe(401);
  });

  it("handles Prometheus NaN histogram quantiles gracefully instead of crashing with 503", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(async (url: RequestInfo | URL) => {
        const u = String(url);
        if (u.includes("histogram_quantile")) {
          return new Response(
            JSON.stringify({ status: "success", data: { result: [{ metric: {}, value: [1700000000, "NaN"] }] } }),
            { status: 200 },
          );
        }
        return new Response(JSON.stringify({ status: "success", data: { result: [] } }), { status: 200 });
      }),
    );

    const response = await getMetrics();
    expect(response.status).toBe(200);
    const data = await response.json();
    expect(data.fleet_latency_p95_ms).toBeNull();
  });

  it("returns source unavailable instead of fabricated metrics", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("offline")));
    const response = await getMetrics();
    expect(response.status).toBe(503);
    expect((await response.json()).source).toBe("prometheus");
  });
  it("returns source unavailable for logs and traces without a backend", async () => {
    expect((await getLogs()).status).toBe(501);
    expect((await getTraces()).status).toBe(501);
  });
  it("maps Prometheus alerts without locally seeded events", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            status: "success",
            data: {
              alerts: [
                {
                  labels: { alertname: "TargetDown", severity: "critical", job: "aegis" },
                  state: "firing",
                  activeAt: "2026-01-01T00:00:00Z",
                  value: "1",
                },
              ],
            },
          }),
          { status: 200 },
        ),
      ),
    );
    const response = await getAlerts();
    expect(response.status).toBe(200);
    expect((await response.json())[0].rule_name).toBe("TargetDown");
  });
  it("exposes Prometheus rules read-only", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            status: "success",
            data: {
              groups: [
                {
                  rules: [
                    {
                      type: "alerting",
                      name: "TargetDown",
                      query: "up == 0",
                      duration: 60,
                      labels: { severity: "critical" },
                      state: "firing",
                    },
                  ],
                },
              ],
            },
          }),
          { status: 200 },
        ),
      ),
    );
    expect((await getAlertRules()).status).toBe(200);
    expect((await createAlertRule()).status).toBe(405);
  });
});
