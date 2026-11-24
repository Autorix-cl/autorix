import { NextRequest, NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";
import { buildInternalAuthHeaders } from "@/lib/api/internal-headers";
import { requireOperator } from "@/lib/auth/session";
import { redactObject } from "@/lib/api/redact";
import type { DiagnosticBundle } from "@/lib/api/schemas/diagnostics";

const BUNDLE_PROBE_TIMEOUT_MS = 3_000;
const ENGINES = ["aegis", "nexus", "themis", "ego", "janus", "vulcan", "hermes"] as const;

interface EngineProbeSnapshot {
  engine: string;
  endpoint: string;
  status: "healthy" | "degraded" | "unreachable";
  http_code: number;
  latency_ms: number;
  error?: string;
}

async function probeEngine(engine: (typeof ENGINES)[number]): Promise<EngineProbeSnapshot> {
  const endpoint = `${getServiceUrl(engine)}/health/ready`;
  const start = performance.now();
  try {
    const res = await fetch(endpoint, {
      cache: "no-store",
      signal: AbortSignal.timeout(BUNDLE_PROBE_TIMEOUT_MS),
    });
    const latency = Math.round((performance.now() - start) * 10) / 10;
    return {
      engine,
      endpoint,
      status: res.status === 200 ? "healthy" : "degraded",
      http_code: res.status,
      latency_ms: latency,
    };
  } catch (err) {
    return {
      engine,
      endpoint,
      status: "unreachable",
      http_code: 0,
      latency_ms: Math.round((performance.now() - start) * 10) / 10,
      error: err instanceof Error ? err.message : "unreachable",
    };
  }
}

async function collectBundlePayload() {
  const createdAt = new Date().toISOString();
  const bundleId = `diag_${createdAt.replace(/[-:TZ.]/g, "").slice(0, 14)}`;

  const headers = buildInternalAuthHeaders();
  const [probes, instancesRes, auditRes] = await Promise.all([
    Promise.all(ENGINES.map(probeEngine)),
    fetch(`${getServiceUrl("argus")}/v1/instances?limit=100`, {
      cache: "no-store",
      headers,
      signal: AbortSignal.timeout(BUNDLE_PROBE_TIMEOUT_MS),
    })
      .then(async (r) => (r.ok ? r.json() : null))
      .catch(() => null),
    fetch(`${getServiceUrl("argus")}/v1/audit?limit=50`, {
      cache: "no-store",
      headers,
      signal: AbortSignal.timeout(BUNDLE_PROBE_TIMEOUT_MS),
    })
      .then(async (r) => (r.ok ? r.json() : null))
      .catch(() => null),
  ]);

  const instances = Array.isArray(instancesRes)
    ? instancesRes
    : Array.isArray(instancesRes?.data)
      ? instancesRes.data
      : [];
  const auditEvents = Array.isArray(auditRes) ? auditRes : Array.isArray(auditRes?.data) ? auditRes.data : [];

  const healthyEngines = probes.filter((p) => p.status === "healthy").length;
  const redactedSnapshot = redactObject({
    collected_at: createdAt,
    probes,
    instances,
    recent_audit_events: auditEvents,
  });

  const summary: DiagnosticBundle & { bundle: typeof redactedSnapshot } = {
    bundle_id: bundleId,
    created_at: createdAt,
    fleet_summary: {
      total_engines: ENGINES.length,
      healthy_engines: healthyEngines,
      total_instances: instances.length || healthyEngines,
    },
    engines_included: [...ENGINES],
    events_count: auditEvents.length,
    logs_count: 0,
    redacted: true,
    download_url: "/api/diagnostics/bundle?download=1",
    bundle: redactedSnapshot,
  };

  return summary;
}

export async function GET(request: NextRequest) {
  const auth = await requireOperator("diagnostics:read");
  if (!auth.ok) return auth.response;

  const summary = await collectBundlePayload();
  const wantsDownload = new URL(request.url).searchParams.get("download") === "1";

  if (wantsDownload) {
    return new NextResponse(JSON.stringify(summary, null, 2), {
      status: 200,
      headers: {
        "Content-Type": "application/json; charset=utf-8",
        "Content-Disposition": 'attachment; filename="autorix-diagnostic-bundle.json"',
      },
    });
  }

  return NextResponse.json(summary, { status: 200 });
}

export async function POST(request: NextRequest) {
  const auth = await requireOperator("diagnostics:read");
  if (!auth.ok) return auth.response;

  const summary = await collectBundlePayload();
  const wantsDownload = new URL(request.url).searchParams.get("download") === "1";

  if (wantsDownload) {
    return new NextResponse(JSON.stringify(summary, null, 2), {
      status: 200,
      headers: {
        "Content-Type": "application/json; charset=utf-8",
        "Content-Disposition": 'attachment; filename="autorix-diagnostic-bundle.json"',
      },
    });
  }

  return NextResponse.json(summary, { status: 201 });
}
