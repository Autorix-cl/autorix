import { NextResponse } from "next/server";
import { createHash } from "crypto";
import { getPrometheusUrl, getServiceUrl } from "@/lib/api-config";
import { buildInternalAuthHeaders } from "@/lib/api/internal-headers";
import { requireOperator } from "@/lib/auth/session";

const NOTIFICATIONS_TIMEOUT_MS = 5_000;

export interface FleetNotificationItem {
  id: string;
  type: "security" | "success" | "info";
  title: string;
  message: string;
  timestamp: string;
}

function deterministicAuditId(action: string, engine: string, timestamp: string, actor: string): string {
  const digest = createHash("sha256").update(`${action}|${engine}|${timestamp}|${actor}`).digest("hex").slice(0, 12);
  return `audit-${digest}`;
}

export async function GET() {
  const auth = await requireOperator("fleet:read");
  if (!auth.ok) return auth.response;

  try {
    const argusUrl = getServiceUrl("argus");
    const prometheusUrl = getPrometheusUrl();
    const headers = buildInternalAuthHeaders();

    // Fetch enrollment audit logs, fleet instances, and optional Prometheus alerts in parallel
    const [auditRes, instancesRes, alertsRes] = await Promise.allSettled([
      fetch(`${argusUrl}/v1/enrollment-audit?limit=25`, {
        cache: "no-store",
        headers,
        signal: AbortSignal.timeout(NOTIFICATIONS_TIMEOUT_MS),
      }),
      fetch(`${argusUrl}/v1/instances?limit=25`, {
        cache: "no-store",
        headers,
        signal: AbortSignal.timeout(NOTIFICATIONS_TIMEOUT_MS),
      }),
      prometheusUrl
        ? fetch(`${prometheusUrl}/api/v1/alerts`, {
            cache: "no-store",
            signal: AbortSignal.timeout(NOTIFICATIONS_TIMEOUT_MS),
          })
        : Promise.resolve(null),
    ]);

    const notifications: FleetNotificationItem[] = [];

    // Parse audit logs if available
    if (auditRes.status === "fulfilled") {
      if (auditRes.value.ok) {
        const auditData = await auditRes.value.json();
        const entries = Array.isArray(auditData?.data) ? auditData.data : [];

        for (const entry of entries) {
          const time = entry.occurred_at || new Date(0).toISOString();
          const actor = entry.actor || "Operator";
          const engine = (entry.engine_type || "Engine").toUpperCase();
          const action = String(entry.action || "unknown");
          const entryId =
            entry.id || entry.token_id
              ? `audit-${entry.id || entry.token_id}`
              : deterministicAuditId(action, engine, time, actor);

          switch (entry.action) {
            case "mint":
              notifications.push({
                id: entryId,
                type: "info",
                title: `Enrollment Token Minted (${engine})`,
                message: `Token for ${engine} minted by ${actor}`,
                timestamp: time,
              });
              break;
            case "consume":
              notifications.push({
                id: entryId,
                type: "success",
                title: `Engine Joined Cluster (${engine})`,
                message: `Instance successfully enrolled in control plane`,
                timestamp: time,
              });
              break;
            case "consume_failed":
              notifications.push({
                id: entryId,
                type: "security",
                title: `Enrollment Rejected (${engine})`,
                message: `Registration attempt rejected: ${entry.detail?.reason || "invalid or exhausted token"}`,
                timestamp: time,
              });
              break;
            case "revoke":
              notifications.push({
                id: entryId,
                type: "security",
                title: `Enrollment Token Revoked (${engine})`,
                message: `Token for ${engine} was revoked`,
                timestamp: time,
              });
              break;
            default:
              notifications.push({
                id: entryId,
                type: "info",
                title: `Fleet Event (${engine})`,
                message: `Action ${action} recorded`,
                timestamp: time,
              });
          }
        }
      } else {
        await auditRes.value.body?.cancel().catch(() => {});
      }
    }

    // Parse instance anomalies or statuses if any
    if (instancesRes.status === "fulfilled") {
      if (instancesRes.value.ok) {
        const instData = await instancesRes.value.json();
        const instances = Array.isArray(instData?.data) ? instData.data : [];

        for (const inst of instances) {
          const engine = (inst.engine_type || inst.type || "Engine").toUpperCase();
          const shortId = (inst.id || "").slice(0, 8);
          if (inst.status === "unreachable" || inst.status === "degraded") {
            notifications.push({
              id: `inst-${inst.id}-${inst.status}`,
              type: "security",
              title: `Engine ${engine} is ${inst.status.toUpperCase()}`,
              message: `Instance ${shortId} missed heartbeats in ${inst.environment || "cluster"}`,
              timestamp: inst.last_heartbeat_at || new Date().toISOString(),
            });
          }
        }
      } else {
        await instancesRes.value.body?.cancel().catch(() => {});
      }
    }

    // Prometheus alerts are also displayed in the inbox. Delivery remains Alertmanager-owned.
    if (alertsRes.status === "fulfilled" && alertsRes.value) {
      if (alertsRes.value.ok) {
        const alertsData = await alertsRes.value.json();
        const alerts = Array.isArray(alertsData?.data?.alerts) ? alertsData.data.alerts : [];
        for (const alert of alerts) {
          if (alert.state !== "firing") continue;
          const labels = alert.labels || {};
          const name = labels.alertname || "Prometheus alert";
          notifications.push({
            id: `prometheus-${[name, labels.instance, alert.activeAt].filter(Boolean).join("-")}`,
            type: labels.severity === "critical" ? "security" : "info",
            title: name,
            message: labels.summary || labels.description || `Prometheus reports ${name} as firing`,
            timestamp: alert.activeAt || new Date(0).toISOString(),
          });
        }
      } else {
        await alertsRes.value.body?.cancel().catch(() => {});
      }
    }

    // Sort descending by timestamp
    notifications.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

    return NextResponse.json({ notifications });
  } catch (err: unknown) {
    return NextResponse.json(
      { notifications: [], error: err instanceof Error ? err.message : "Failed to fetch notifications" },
      { status: 500 },
    );
  }
}
