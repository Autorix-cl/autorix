import { NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";
import { requireOperator } from "@/lib/auth/session";
import type { ChangeTimelineEvent } from "@/lib/api/schemas/diagnostics";

const TIMELINE_TIMEOUT_MS = 5_000;

function mapAuditEventType(action: string, resourceType: string): ChangeTimelineEvent["event_type"] {
  const combined = `${action} ${resourceType}`.toLowerCase();
  if (combined.includes("migration") || combined.includes("schema")) return "schema_migration";
  if (combined.includes("enroll") || combined.includes("register") || combined.includes("instance")) return "registration";
  if (combined.includes("deploy")) return "deployment";
  return "config_change";
}

export async function GET() {
  const auth = await requireOperator("diagnostics:read");
  if (!auth.ok) return auth.response;

  const argusUrl = getServiceUrl("argus");
  const [auditRes, enrollmentRes] = await Promise.allSettled([
    fetch(`${argusUrl}/v1/audit?limit=50`, {
      cache: "no-store",
      signal: AbortSignal.timeout(TIMELINE_TIMEOUT_MS),
    }),
    fetch(`${argusUrl}/v1/enrollment-audit?limit=25`, {
      cache: "no-store",
      signal: AbortSignal.timeout(TIMELINE_TIMEOUT_MS),
    }),
  ]);

  const auditOk = auditRes.status === "fulfilled" && auditRes.value.ok;
  const enrollOk = enrollmentRes.status === "fulfilled" && enrollmentRes.value.ok;

  if (!auditOk && !enrollOk) {
    return NextResponse.json(
      {
        error: "Argus audit timeline source unavailable",
        source: "unavailable",
        events: [],
      },
      { status: 503, headers: { "X-Autorix-Source": "unavailable" } },
    );
  }

  const events: ChangeTimelineEvent[] = [];

  if (auditOk) {
    const auditBody = await auditRes.value.json().catch(() => null);
    const records = Array.isArray(auditBody) ? auditBody : Array.isArray(auditBody?.data) ? auditBody.data : [];
    for (const rec of records as Record<string, unknown>[]) {
      const action = String(rec.action || "change");
      const resourceType = String(rec.resource_type || "config");
      const engineType = String(rec.engine_type || rec.service || resourceType || "argus");
      const outcome = String(rec.outcome || "success").toLowerCase();
      events.push({
        id: String(rec.id || `audit-${rec.created_at || rec.occurred_at || events.length}`),
        timestamp: String(rec.occurred_at || rec.created_at || new Date(0).toISOString()),
        event_type: mapAuditEventType(action, resourceType),
        engine_type: engineType,
        title: `${action.toUpperCase()} ${resourceType}`,
        description: String(rec.description || `${action} on ${resourceType} (${String(rec.resource_id || "global")})`),
        actor: String(rec.actor || "system"),
        correlated_error_spike: outcome === "failure" || outcome === "error",
      });
    }
  }

  if (enrollOk) {
    const enrollBody = await enrollmentRes.value.json().catch(() => null);
    const records = Array.isArray(enrollBody) ? enrollBody : Array.isArray(enrollBody?.data) ? enrollBody.data : [];
    for (const rec of records as Record<string, unknown>[]) {
      const action = String(rec.action || "enrollment");
      const engineType = String(rec.engine_type || "argus");
      events.push({
        id: String(rec.id || `enroll-${rec.occurred_at || events.length}`),
        timestamp: String(rec.occurred_at || new Date(0).toISOString()),
        event_type: "registration",
        engine_type: engineType,
        title: `Enrollment ${action} (${engineType})`,
        description: `Enrollment action '${action}' recorded for ${engineType}`,
        actor: String(rec.actor || "system"),
        correlated_error_spike: action === "consume_failed",
      });
    }
  }

  events.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

  return NextResponse.json(events, {
    status: 200,
    headers: { "X-Autorix-Source": "live" },
  });
}
