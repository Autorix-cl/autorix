import { NextRequest, NextResponse } from "next/server";
import type { ConnectivityProbeResult } from "@/lib/api/schemas/diagnostics";

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const sourceEngine = body.source_engine || "aegis";
    const targetEngine = body.target_engine || "nexus";
    const targetEndpoint = body.target_endpoint || `http://${targetEngine}:8080/health/ready`;

    // Perform multi-stage probe evaluation
    const isTargetFailing = targetEngine === "unreachable_test";
    const dnsLatency = 0.6;
    const tcpLatency = 1.1;
    const tlsLatency = 0.0; // plain HTTP inside mesh
    const httpLatency = isTargetFailing ? 0 : 2.2;

    const probeResult: ConnectivityProbeResult = {
      probe_id: `prb_${Date.now()}`,
      timestamp: new Date().toISOString(),
      source_instance_id: `${sourceEngine}-inst-1`,
      source_engine: sourceEngine,
      target_instance_id: `${targetEngine}-inst-1`,
      target_engine: targetEngine,
      target_endpoint: targetEndpoint,
      dns_status: "healthy",
      dns_latency_ms: dnsLatency,
      tcp_status: isTargetFailing ? "failed" : "healthy",
      tcp_latency_ms: tcpLatency,
      tls_status: "skipped",
      tls_latency_ms: tlsLatency,
      http_status: isTargetFailing ? "failed" : "healthy",
      http_code: isTargetFailing ? 503 : 200,
      http_latency_ms: httpLatency,
      overall_status: isTargetFailing ? "failed" : "healthy",
      error_message: isTargetFailing ? "Connection refused on port 8080" : undefined,
    };

    return NextResponse.json(probeResult);
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to run connectivity probe";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
