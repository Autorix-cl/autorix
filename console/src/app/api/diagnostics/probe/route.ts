import { NextRequest, NextResponse } from "next/server";
import { BACKEND_URLS, getServiceUrl } from "@/lib/api-config";
import { requireOperator } from "@/lib/auth/session";
import type { ConnectivityProbeResult } from "@/lib/api/schemas/diagnostics";

const PROBE_TIMEOUT_MS = 5_000;
type ServiceKey = keyof typeof BACKEND_URLS;

function isRegisteredEngine(name: unknown): name is ServiceKey {
  return typeof name === "string" && Object.prototype.hasOwnProperty.call(BACKEND_URLS, name);
}

function roundMs(ms: number): number {
  return Math.round(ms * 10) / 10;
}

export async function POST(req: NextRequest) {
  const auth = await requireOperator("diagnostics:read");
  if (!auth.ok) return auth.response;

  try {
    const body = await req.json();
    const sourceEngine = typeof body.source_engine === "string" && body.source_engine ? body.source_engine : "aegis";
    const targetEngine = body.target_engine || "nexus";

    if (!isRegisteredEngine(targetEngine)) {
      return NextResponse.json(
        { error: `Invalid target engine '${String(targetEngine)}': must be a registered service` },
        { status: 400 },
      );
    }

    const baseUrl = getServiceUrl(targetEngine);
    const targetEndpoint = `${baseUrl}/health/ready`;
    const isHttps = targetEndpoint.startsWith("https://");

    const start = performance.now();
    try {
      const res = await fetch(targetEndpoint, {
        method: "GET",
        cache: "no-store",
        signal: AbortSignal.timeout(PROBE_TIMEOUT_MS),
      });
      const totalMs = roundMs(performance.now() - start);
      const isHealthy = res.status === 200;
      const isDegraded = res.status === 503 || (res.status >= 400 && res.status < 500);
      const httpStatus = isHealthy ? "healthy" : isDegraded ? "degraded" : "failed";
      const overallStatus = isHealthy ? "healthy" : isDegraded ? "degraded" : "failed";

      const probeResult: ConnectivityProbeResult = {
        probe_id: `prb_${Date.now()}`,
        timestamp: new Date().toISOString(),
        source_instance_id: `${sourceEngine}-console-probe`,
        source_engine: sourceEngine,
        target_instance_id: `${targetEngine}-service`,
        target_engine: targetEngine,
        target_endpoint: targetEndpoint,
        dns_status: "healthy",
        dns_latency_ms: 0,
        tcp_status: "healthy",
        tcp_latency_ms: 0,
        tls_status: isHttps ? "healthy" : "skipped",
        tls_latency_ms: 0,
        http_status: httpStatus,
        http_code: res.status,
        http_latency_ms: totalMs,
        overall_status: overallStatus,
        error_message: isHealthy ? undefined : `Target returned HTTP ${res.status}`,
      };

      return NextResponse.json(probeResult);
    } catch (err) {
      const elapsedMs = roundMs(performance.now() - start);
      const errorMessage = err instanceof Error ? err.message : "Target engine unreachable";

      const probeResult: ConnectivityProbeResult = {
        probe_id: `prb_${Date.now()}`,
        timestamp: new Date().toISOString(),
        source_instance_id: `${sourceEngine}-console-probe`,
        source_engine: sourceEngine,
        target_instance_id: `${targetEngine}-service`,
        target_engine: targetEngine,
        target_endpoint: targetEndpoint,
        dns_status: "failed",
        dns_latency_ms: 0,
        tcp_status: "failed",
        tcp_latency_ms: elapsedMs,
        tls_status: isHttps ? "failed" : "skipped",
        tls_latency_ms: 0,
        http_status: "failed",
        http_code: 0,
        http_latency_ms: 0,
        overall_status: "failed",
        error_message: errorMessage,
      };

      return NextResponse.json(probeResult);
    }
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to run connectivity probe";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
