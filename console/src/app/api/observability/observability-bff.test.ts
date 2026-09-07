import { describe, it, expect, vi, beforeEach } from "vitest";
import { GET as getMetrics } from "./metrics/route";
import { GET as getTraces } from "./traces/route";
import { GET as getLogs } from "./logs/route";
import { GET as getAlerts } from "./alerts/route";
import { GET as getAlertRules, POST as createAlertRule } from "./alerts/rules/route";

describe("Observability BFF routes", () => {
  beforeEach(() => vi.restoreAllMocks());
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
      vi
        .fn()
        .mockResolvedValue(
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
      vi
        .fn()
        .mockResolvedValue(
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
