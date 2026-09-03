import { describe, it, expect, vi, beforeEach } from "vitest";
import { GET as getMetrics } from "./metrics/route";
import { GET as getLogs } from "./logs/route";
import { GET as getTraces } from "./traces/route";
import { GET as getTraceDetail } from "./traces/[id]/route";
import { GET as getAlerts } from "./alerts/route";
import { GET as getAlertRules, POST as createAlertRule } from "./alerts/rules/route";
import { GET as getSLOs } from "./slo/route";
import { NextRequest } from "next/server";

describe("Observability BFF Routes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("GET /api/observability/metrics returns fleet metrics summary", async () => {
    const res = await getMetrics();
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data.total_engines).toBeGreaterThanOrEqual(7);
    expect(data.fleet_qps).toBeGreaterThanOrEqual(0);
    expect(data.engines.length).toBeGreaterThanOrEqual(7);
  });

  it("GET /api/observability/logs supports filtering by engine and level", async () => {
    const req = new NextRequest("http://localhost/api/observability/logs?engine=aegis&level=info");
    const res = await getLogs(req);
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(Array.isArray(data)).toBe(true);
    expect(data.every((l: { engine: string }) => l.engine === "aegis")).toBe(true);
  });

  it("GET /api/observability/traces returns traces and detail", async () => {
    const res = await getTraces();
    expect(res.status).toBe(200);
    const list = await res.json();
    expect(list.length).toBeGreaterThan(0);

    const detailReq = new NextRequest("http://localhost/api/observability/traces/tr-7710a");
    const detailRes = await getTraceDetail(detailReq, { params: Promise.resolve({ id: "tr-7710a" }) });
    expect(detailRes.status).toBe(200);
    const detail = await detailRes.json();
    expect(detail.trace_id).toBe("tr-7710a");
    expect(detail.spans.length).toBeGreaterThanOrEqual(3);
  });

  it("GET & POST /api/observability/alerts/rules manages alert rules", async () => {
    const listRes = await getAlertRules();
    expect(listRes.status).toBe(200);
    const rules = await listRes.json();
    expect(rules.length).toBeGreaterThan(0);

    const postReq = new NextRequest("http://localhost/api/observability/alerts/rules", {
      method: "POST",
      body: JSON.stringify({
        name: "Test High Latency Rule",
        engine_type: "nexus",
        severity: "warning",
        metric: "nexus_p95_ms",
        threshold: 20,
      }),
    });
    const postRes = await createAlertRule(postReq);
    expect(postRes.status).toBe(201);
    const created = await postRes.json();
    expect(created.name).toBe("Test High Latency Rule");
  });

  it("GET /api/observability/alerts and /slo return status and error budgets", async () => {
    const alertsRes = await getAlerts();
    expect(alertsRes.status).toBe(200);
    const alerts = await alertsRes.json();
    expect(alerts.length).toBeGreaterThan(0);

    const sloRes = await getSLOs();
    expect(sloRes.status).toBe(200);
    const slos = await sloRes.json();
    expect(slos.length).toBeGreaterThan(0);
    expect(slos[0].target_percentage).toBeGreaterThan(90);
  });
});
