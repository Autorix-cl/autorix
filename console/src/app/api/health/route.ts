/**
 * Honest health aggregator (P1-S4-T1). Replaces a route whose catch branch
 * unconditionally returned "healthy" — the dashboard used to report 7/7
 * healthy with every engine stopped. Every one of the 7 engines now serves
 * a real GET /health/ready (P1-S1), so this probes that instead of an
 * arbitrary business endpoint, and never claims a state it did not measure.
 */
import { NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";

const PROBE_TIMEOUT_MS = 2000;

type EngineStatus = "healthy" | "degraded" | "unreachable" | "unknown";

interface ServiceDescriptor {
  key: Parameters<typeof getServiceUrl>[0];
  name: string;
  port: number;
  protocol: string;
}

interface ServiceHealth extends ServiceDescriptor {
  status: EngineStatus;
  latencyMs: number | null;
}

// Ports match each engine's health-contract listener (P1-S1 ADR): aegis and
// nexus serve /health/ready on their admin/REST port, never on aegis's
// proxy port (4455) or nexus's gRPC port (50051).
const SERVICES: ServiceDescriptor[] = [
  { key: "ego", name: "Autorix Ego", port: 4433, protocol: "REST" },
  { key: "janus", name: "Autorix Janus", port: 4444, protocol: "REST/OIDC" },
  { key: "vulcan", name: "Autorix Vulcan", port: 4466, protocol: "REST" },
  { key: "hermes", name: "Autorix Hermes", port: 4477, protocol: "REST/XML" },
  { key: "themis", name: "Autorix Themis", port: 4488, protocol: "REST/gRPC" },
  { key: "aegis", name: "Autorix Aegis", port: 4456, protocol: "HTTP Proxy (admin)" },
  { key: "nexus", name: "Autorix Nexus", port: 8080, protocol: "gRPC (REST admin)" },
];

async function probe(service: ServiceDescriptor): Promise<ServiceHealth> {
  const url = `${getServiceUrl(service.key)}/health/ready`;
  const start = Date.now();
  try {
    const res = await fetch(url, { cache: "no-store", signal: AbortSignal.timeout(PROBE_TIMEOUT_MS) });
    const latencyMs = Date.now() - start;
    let status: EngineStatus;
    if (res.status === 200) status = "healthy";
    else if (res.status === 503) status = "degraded";
    else status = "unknown";
    return { ...service, status, latencyMs };
  } catch {
    // Network failure, timeout, DNS — the engine could not be reached at
    // all, distinct from a 503 (reachable, but reporting not ready).
    return { ...service, status: "unreachable", latencyMs: null };
  }
}

export async function GET() {
  const services = await Promise.all(SERVICES.map(probe));
  return NextResponse.json({
    checkedAt: new Date().toISOString(),
    services,
  });
}
