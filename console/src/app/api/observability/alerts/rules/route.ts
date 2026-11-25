import { NextResponse } from "next/server";
import { getPrometheusUrl } from "@/lib/api-config";
import { requireOperator } from "@/lib/auth/session";
import { alertRuleListSchema } from "@/lib/api/schemas/observability";

const PROMETHEUS_TIMEOUT_MS = 8_000;

export async function GET() {
  const auth = await requireOperator("observability:read");
  if (!auth.ok) return auth.response;

  const prometheusUrl = getPrometheusUrl();
  if (!prometheusUrl) {
    return NextResponse.json(
      {
        error: "Prometheus alert rules are not configured",
        source: "prometheus",
      },
      { status: 503 },
    );
  }

  try {
    const response = await fetch(`${prometheusUrl}/api/v1/rules`, {
      cache: "no-store",
      signal: AbortSignal.timeout(PROMETHEUS_TIMEOUT_MS),
    });
    if (!response.ok) {
      await response.body?.cancel().catch(() => {});
      throw new Error(`Prometheus returned ${response.status}`);
    }
    const body = await response.json();
    if (body.status !== "success" || !Array.isArray(body.data?.groups))
      throw new Error("Prometheus returned an invalid rules response");
    const rules = body.data.groups.flatMap(
      (group: {
        rules?: Array<{
          type?: string;
          name?: string;
          query?: string;
          duration?: number;
          labels?: Record<string, string>;
          state?: string;
        }>;
      }) =>
        (group.rules ?? [])
          .filter((rule) => rule.type === "alerting")
          .map((rule) => ({
            id: rule.name ?? "unknown",
            name: rule.name ?? "Unnamed alert",
            engine_type: rule.labels?.engine ?? "unknown",
            severity:
              rule.labels?.severity === "critical" ? "critical" : rule.labels?.severity === "info" ? "info" : "warning",
            metric: rule.query ?? "",
            threshold: null,
            operator: null,
            duration: `${rule.duration ?? 0}s`,
            enabled: rule.state !== "inactive",
          })),
    );
    return NextResponse.json(alertRuleListSchema.parse(rules));
  } catch (error) {
    return NextResponse.json(
      {
        error: "Prometheus alert rules are unavailable",
        source: "prometheus",
        detail: error instanceof Error ? error.message : undefined,
      },
      { status: 503 },
    );
  }
}
export async function POST() {
  const auth = await requireOperator("observability:read");
  if (!auth.ok) return auth.response;

  return NextResponse.json(
    { error: "Alert rules are managed by Prometheus configuration" },
    { status: 405, headers: { Allow: "GET" } },
  );
}
