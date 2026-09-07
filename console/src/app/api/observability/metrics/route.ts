import { NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";
import { fleetMetricsSummarySchema, type EngineMetricSummary } from "@/lib/api/schemas/observability";

interface PrometheusResult { metric: Record<string, string>; value: [number, string]; }

async function query(query: string): Promise<PrometheusResult[]> {
  const response = await fetch(`${getServiceUrl("prometheus")}/api/v1/query?query=${encodeURIComponent(query)}`, { cache: "no-store" });
  if (!response.ok) throw new Error(`Prometheus returned ${response.status}`);
  const body = await response.json();
  if (body.status !== "success" || !Array.isArray(body.data?.result)) throw new Error("Prometheus returned an invalid query response");
  return body.data.result;
}
function scalar(results: PrometheusResult[]): number { return Number(results[0]?.value?.[1] ?? 0); }
function round(value: number): number { return Math.round(value * 100) / 100; }

export async function GET() {
  try {
    const [requests, qps, errors, p95, up, byEngine, qpsByEngine] = await Promise.all([
      query("sum(autorix_http_requests_total)"),
      query("sum(rate(autorix_http_requests_total[1m]))"),
      query('sum(rate(autorix_http_requests_total{status=~"5.."}[1m]))'),
      query("histogram_quantile(0.95, sum by (le) (rate(autorix_http_request_duration_seconds_bucket[5m]))) * 1000"),
      query("up{job=~\".*autorix.*\"}"),
      query("sum by (engine) (autorix_http_requests_total)"),
      query("sum by (engine) (rate(autorix_http_requests_total[1m]))"),
    ]);
    const totalRequests = scalar(requests);
    const fleetQps = scalar(qps);
    const errorQps = scalar(errors);
    const engines: EngineMetricSummary[] = byEngine.map((item) => {
      const engine = item.metric.engine;
      const engineQps = scalar(qpsByEngine.filter((candidate) => candidate.metric.engine === engine));
      return {
        engine_type: engine,
        status: "observed",
        instance_count: up.filter((target) => target.metric.engine === engine && target.value[1] === "1").length,
        requests_total: Number(item.value[1]), requests_per_second: round(engineQps), error_rate: null,
        latency_p50_ms: null, latency_p95_ms: null, latency_p99_ms: null,
        auth_decisions_total: null, auth_allow_rate: null,
      };
    });
    return NextResponse.json(fleetMetricsSummarySchema.parse({
      timestamp: new Date().toISOString(), total_engines: engines.length, total_instances: up.length,
      healthy_instances: up.filter((target) => target.value[1] === "1").length,
      requests_total: totalRequests, fleet_qps: round(fleetQps),
      fleet_error_rate: fleetQps > 0 ? errorQps / fleetQps : 0,
      fleet_latency_p95_ms: p95.length ? round(scalar(p95)) : null,
      auth_decisions_total: null, auth_allow_rate: null, engines,
    }));
  } catch (error) {
    return NextResponse.json({ error: "Prometheus metrics are unavailable", source: "prometheus", detail: error instanceof Error ? error.message : undefined }, { status: 503 });
  }
}
