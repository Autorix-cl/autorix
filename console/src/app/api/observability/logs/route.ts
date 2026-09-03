import { NextRequest, NextResponse } from "next/server";
import type { LogEntry } from "@/lib/api/schemas/observability";

const SAMPLE_LOGS: LogEntry[] = [
  {
    id: "log-1",
    timestamp: new Date(Date.now() - 2000).toISOString(),
    engine: "aegis",
    instance_id: "aegis-inst-1",
    level: "info",
    message: "HTTP Ingress: Route match found for /api/v1/projects/alpha/deployments",
    request_id: "req-98fbc1",
    correlation_id: "corr-98fbc1",
    trace_id: "tr-7710a",
    attributes: { method: "GET", client_ip: "10.0.4.12", upstream: "http://deploy-engine:8080" },
  },
  {
    id: "log-2",
    timestamp: new Date(Date.now() - 1800).toISOString(),
    engine: "vulcan",
    instance_id: "vulcan-inst-1",
    level: "info",
    message: "API Key verified successfully: prefix av_live_78ab, scopes: [read, deploy]",
    request_id: "req-98fbc1",
    correlation_id: "corr-98fbc1",
    trace_id: "tr-7710a",
    attributes: { key_id: "vk_7721", latency_us: 420 },
  },
  {
    id: "log-3",
    timestamp: new Date(Date.now() - 1500).toISOString(),
    engine: "nexus",
    instance_id: "nexus-inst-1",
    level: "info",
    message: "Zanzibar Check: user:operator has relation 'operator' on project:proj_alpha",
    request_id: "req-98fbc1",
    correlation_id: "corr-98fbc1",
    trace_id: "tr-7710a",
    attributes: { allowed: true, cache_hit: true, depth: 2 },
  },
  {
    id: "log-4",
    timestamp: new Date(Date.now() - 1200).toISOString(),
    engine: "themis",
    instance_id: "themis-inst-1",
    level: "info",
    message: "ABAC Evaluation: Policy 'allow-devops-in-region' passed",
    request_id: "req-98fbc1",
    correlation_id: "corr-98fbc1",
    trace_id: "tr-7710a",
    attributes: { duration_us: 210, policies_evaluated: 1 },
  },
  {
    id: "log-5",
    timestamp: new Date(Date.now() - 900).toISOString(),
    engine: "aegis",
    instance_id: "aegis-inst-1",
    level: "info",
    message: "HTTP Upstream Response: 200 OK (latency: 4.2ms)",
    request_id: "req-98fbc1",
    correlation_id: "corr-98fbc1",
    trace_id: "tr-7710a",
    attributes: { status: 200, bytes_sent: 1420 },
  },
  {
    id: "log-6",
    timestamp: new Date(Date.now() - 15000).toISOString(),
    engine: "ego",
    instance_id: "ego-inst-1",
    level: "warn",
    message: "Failed password attempt for user: john.doe@example.com (locked: false)",
    request_id: "req-1122a",
    correlation_id: "corr-1122a",
    trace_id: "tr-2219b",
    attributes: { ip: "192.168.1.105", attempt: 2 },
  },
  {
    id: "log-7",
    timestamp: new Date(Date.now() - 45000).toISOString(),
    engine: "hermes",
    instance_id: "hermes-inst-1",
    level: "error",
    message: "SAML assertion signature verification failed: invalid certificate signature",
    request_id: "req-3388c",
    correlation_id: "corr-3388c",
    trace_id: "tr-4411c",
    attributes: { idp_entity: "https://idp.badcert.com", error_code: "CERT_MISMATCH" },
  },
];

export async function GET(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const engine = searchParams.get("engine");
  const level = searchParams.get("level");
  const requestId = searchParams.get("request_id");
  const query = searchParams.get("q")?.toLowerCase();

  let filtered = [...SAMPLE_LOGS];

  if (engine && engine !== "all") {
    filtered = filtered.filter((l) => l.engine === engine);
  }
  if (level && level !== "all") {
    filtered = filtered.filter((l) => l.level === level);
  }
  if (requestId) {
    filtered = filtered.filter((l) => l.request_id === requestId || l.correlation_id === requestId);
  }
  if (query) {
    filtered = filtered.filter(
      (l) =>
        l.message.toLowerCase().includes(query) ||
        l.engine.toLowerCase().includes(query) ||
        l.request_id?.toLowerCase().includes(query)
    );
  }

  return NextResponse.json(filtered);
}
