import { NextRequest, NextResponse } from "next/server";
import type { TraceDetail } from "@/lib/api/schemas/observability";

type Params = { params: Promise<{ id: string }> };

export async function GET(req: NextRequest, { params }: Params) {
  const { id } = await params;

  // Standard detailed trace
  const trace: TraceDetail = {
    trace_id: id,
    root_operation: "HTTP GET /api/v1/projects/alpha/deployments",
    root_engine: "aegis",
    total_duration_ms: 4.6,
    timestamp: new Date().toISOString(),
    status: "ok",
    spans: [
      {
        span_id: "sp-root",
        trace_id: id,
        engine: "aegis",
        operation: "ingress_route_match",
        status: "ok",
        start_time_ms: 0,
        duration_ms: 4.6,
        attributes: { route: "/api/v1/projects/*", upstream: "http://deploy-engine:8080" },
      },
      {
        span_id: "sp-vulcan",
        parent_span_id: "sp-root",
        trace_id: id,
        engine: "vulcan",
        operation: "verify_api_key",
        status: "ok",
        start_time_ms: 0.6,
        duration_ms: 0.8,
        attributes: { prefix: "av_live_78ab", scopes: ["read", "deploy"] },
      },
      {
        span_id: "sp-nexus",
        parent_span_id: "sp-root",
        trace_id: id,
        engine: "nexus",
        operation: "check_relation",
        status: "ok",
        start_time_ms: 1.5,
        duration_ms: 1.8,
        attributes: { subject: "user:operator", object: "project:proj_alpha", relation: "operator" },
      },
      {
        span_id: "sp-themis",
        parent_span_id: "sp-root",
        trace_id: id,
        engine: "themis",
        operation: "evaluate_policy",
        status: "ok",
        start_time_ms: 3.4,
        duration_ms: 0.6,
        attributes: { policy: "allow-devops-in-region", passed: true },
      },
      {
        span_id: "sp-upstream",
        parent_span_id: "sp-root",
        trace_id: id,
        engine: "aegis",
        operation: "upstream_proxy_roundtrip",
        status: "ok",
        start_time_ms: 4.0,
        duration_ms: 0.6,
        attributes: { http_status: 200 },
      },
    ],
  };

  return NextResponse.json(trace);
}
