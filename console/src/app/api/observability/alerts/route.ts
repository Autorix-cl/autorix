import { NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";
import { alertEventListSchema } from "@/lib/api/schemas/observability";

export async function GET() {
  try {
    const response = await fetch(`${getServiceUrl("prometheus")}/api/v1/alerts`, { cache: "no-store" });
    if (!response.ok) throw new Error(`Prometheus returned ${response.status}`);
    const body = await response.json();
    if (body.status !== "success" || !Array.isArray(body.data?.alerts))
      throw new Error("Prometheus returned an invalid alerts response");
    const alerts = body.data.alerts.map(
      (alert: { labels?: Record<string, string>; state?: string; activeAt?: string; value?: string }) => ({
        id: [alert.labels?.alertname, alert.labels?.instance, alert.activeAt].filter(Boolean).join(":"),
        rule_id: alert.labels?.alertname ?? "unknown",
        rule_name: alert.labels?.alertname ?? "Unnamed alert",
        engine_type: alert.labels?.engine ?? alert.labels?.job ?? "unknown",
        severity:
          alert.labels?.severity === "critical" ? "critical" : alert.labels?.severity === "info" ? "info" : "warning",
        state: alert.state === "firing" ? "firing" : "resolved",
        value: Number(alert.value ?? 0),
        threshold: null,
        triggered_at: alert.activeAt ?? new Date(0).toISOString(),
      }),
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
