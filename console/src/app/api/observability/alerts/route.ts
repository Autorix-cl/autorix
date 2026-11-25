import { NextResponse } from "next/server";
import { getPrometheusUrl } from "@/lib/api-config";
import { requireOperator } from "@/lib/auth/session";
import { alertEventListSchema } from "@/lib/api/schemas/observability";

const PROMETHEUS_TIMEOUT_MS = 8_000;

export async function GET() {
  const auth = await requireOperator("observability:read");
  if (!auth.ok) return auth.response;

  const prometheusUrl = getPrometheusUrl();
  if (!prometheusUrl) {
    return NextResponse.json(
      {
        error: "Prometheus alerts are not configured",
        source: "prometheus",
      },
      { status: 503 },
    );
  }

  try {
    const response = await fetch(`${prometheusUrl}/api/v1/alerts`, {
      cache: "no-store",
      signal: AbortSignal.timeout(PROMETHEUS_TIMEOUT_MS),
    });
    if (!response.ok) {
      await response.body?.cancel().catch(() => {});
      throw new Error(`Prometheus returned ${response.status}`);
    }
    const body = await response.json();
    if (body.status !== "success" || !Array.isArray(body.data?.alerts))
      throw new Error("Prometheus returned an invalid alerts response");
    const alerts = body.data.alerts.map(
      (alert: { labels?: Record<string, string>; state?: string; activeAt?: string; value?: string }) => {
        const parsedValue = Number(alert.value ?? 0);
        return {
          id: [alert.labels?.alertname, alert.labels?.instance, alert.activeAt].filter(Boolean).join(":"),
          rule_id: alert.labels?.alertname ?? "unknown",
          rule_name: alert.labels?.alertname ?? "Unnamed alert",
          engine_type: alert.labels?.engine ?? alert.labels?.job ?? "unknown",
          severity:
            alert.labels?.severity === "critical" ? "critical" : alert.labels?.severity === "info" ? "info" : "warning",
          state: alert.state === "firing" ? "firing" : "resolved",
          value: Number.isFinite(parsedValue) ? parsedValue : 0,
          threshold: null,
          triggered_at: alert.activeAt ?? new Date(0).toISOString(),
        };
      },
    );
    return NextResponse.json(alertEventListSchema.parse(alerts));
  } catch (error) {
    return NextResponse.json(
      {
        error: "Prometheus alerts are unavailable",
        source: "prometheus",
        detail: error instanceof Error ? error.message : undefined,
      },
      { status: 503 },
    );
  }
}
