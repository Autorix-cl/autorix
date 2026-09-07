import { NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";
import { alertRuleListSchema } from "@/lib/api/schemas/observability";

export async function GET() {
  try {
    const response = await fetch(`${getServiceUrl("prometheus")}/api/v1/rules`, { cache: "no-store" });
    if (!response.ok) throw new Error(`Prometheus returned ${response.status}`);
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
  return NextResponse.json(
    { error: "Alert rules are managed by Prometheus configuration" },
    { status: 405, headers: { Allow: "GET" } },
  );
}
