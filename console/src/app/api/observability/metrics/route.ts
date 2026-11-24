import { NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";
import { requireOperator } from "@/lib/auth/session";
import { fleetMetricsSummarySchema, type EngineMetricSummary } from "@/lib/api/schemas/observability";

const PROMETHEUS_TIMEOUT_MS = 8_000;

interface PrometheusResult {
  metric: Record<string, string>;
  value: [number, string];
}

async function query(promQuery: string): Promise<PrometheusResult[]> {
  const response = await fetch(`${getServiceUrl("prometheus")}/api/v1/query?query=${encodeURIComponent(promQuery)}`, {
    cache: "no-store",
    signal: AbortSignal.timeout(PROMETHEUS_TIMEOUT_MS),
  });
  if (!response.ok) throw new Error(`Prometheus returned ${response.status}`);
  const body = await response.json();
  if (body.status !== "success" || !Array.isArray(body.data?.result))
    throw new Error("Prometheus returned an invalid query response");
  return body.data.result;
}

function toFiniteNumber(raw: unknown): number | null {
  if (raw === null || raw === undefined || raw === "") return null;
  const parsed = Number(raw);
  return Number.isFinite(parsed) ? parsed : null;
}

function scalar(results: PrometheusResult[]): number | null {
  return toFiniteNumber(results[0]?.value?.[1]);
}

function round(value: number): number {
  return Math.round(value * 100) / 100;
}

export async function GET() {
  const auth = await requireOperator("observability:read");
  if (!auth.ok) return auth.response;

  try {
    const [requests, qps, errors, p95, up, byEngine, qpsByEngine] = await Promise.all([
      query("sum(autorix_http_requests_total)"),
      query("sum(rate(autorix_http_requests_total[1m]))"),
      query('sum(rate(autorix_http_requests_total{status=~"5.."}[1m]))'),
      query("histogram_quantile(0.95, sum by (le) (rate(autorix_http_request_duration_seconds_bucket[5m]))) * 1000"),
      query('up{job=~".*autorix.*"}'),
      query("sum by (engine) (autorix_http_requests_total)"),
      query("sum by (engine) (rate(autorix_http_requests_total[1m]))"),
    ]);
    const totalRequests = scalar(requests) ?? 0;
    const fleetQps = scalar(qps) ?? 0;
    const errorQps = scalar(errors) ?? 0;
    const p95Value = scalar(p95);

    const engines: EngineMetricSummary[] = byEngine.map((item) => {
      const engine = item.metric.engine || "unknown";
      const engineQps = scalar(qpsByEngine.filter((candidate) => candidate.metric.engine === engine)) ?? 0;
      const engineRequests = toFiniteNumber(item.value?.[1]) ?? 0;
      return {
        engine_type: engine,
        status: "observed",
        instance_count: up.filter((target) => target.metric.engine === engine && target.value[1] === "1").length,
        requests_total: engineRequests,
        requests_per_second: round(engineQps),
        error_rate: null,
        latency_p50_ms: null,
        latency_p95_ms: null,
        latency_p99_ms: null,
        auth_decisions_total: null,
        auth_allow_rate: null,
      };
    });

    const fleetErrorRate = fleetQps > 0 && Number.isFinite(errorQps / fleetQps) ? errorQps / fleetQps : 0;

    return NextResponse.json(
      fleetMetricsSummarySchema.parse({
        timestamp: new Date().toISOString(),
        total_engines: engines.length,
        total_instances: up.length,
        healthy_instances: up.filter((target) => target.value[1] === "1").length,
        requests_total: totalRequests,
        fleet_qps: round(fleetQps),
        fleet_error_rate: fleetErrorRate,
        fleet_latency_p95_ms: p95Value !== null ? round(p95Value) : null,
        auth_decisions_total: null,
        auth_allow_rate: null,
        engines,
      }),
    );
  } catch (error) {
    return NextResponse.json(
      {
        error: "Prometheus metrics are unavailable",
        source: "prometheus",
        detail: error instanceof Error ? error.message : undefined,
      },
      { status: 503 },
    );
  }
}
