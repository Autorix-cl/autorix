import { NextResponse } from "next/server";
import type { TraceDetail } from "@/lib/api/schemas/observability";

const SAMPLE_TRACES: TraceDetail[] = [
  {
    trace_id: "tr-7710a",
    root_operation: "HTTP GET /api/v1/projects/alpha/deployments",
    root_engine: "aegis",
    total_duration_ms: 4.6,
    timestamp: new Date(Date.now() - 2000).toISOString(),
    status: "ok",
    spans: [
      {
        span_id: "sp-root",
        trace_id: "tr-7710a",
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
        trace_id: "tr-7710a",
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
        trace_id: "tr-7710a",
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
        trace_id: "tr-7710a",
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
        trace_id: "tr-7710a",
        engine: "aegis",
        operation: "upstream_proxy_roundtrip",
        status: "ok",
        start_time_ms: 4.0,
        duration_ms: 0.6,
        attributes: { http_status: 200 },
      },
    ],
  },
  {
    trace_id: "tr-2219b",
    root_operation: "POST /v1/auth/login",
    root_engine: "ego",
    total_duration_ms: 82.4,
    timestamp: new Date(Date.now() - 15000).toISOString(),
    status: "error",
    spans: [
      {
        span_id: "sp-ego-root",
        trace_id: "tr-2219b",
        engine: "ego",
        operation: "authenticate_password",
        status: "error",
        start_time_ms: 0,
        duration_ms: 82.4,
        attributes: { email: "john.doe@example.com", error: "invalid credentials" },
      },
      {
        span_id: "sp-argon2",
        parent_span_id: "sp-ego-root",
        trace_id: "tr-2219b",
        engine: "ego",
        operation: "argon2id_verify",
        status: "error",
        start_time_ms: 1.2,
        duration_ms: 81.0,
        attributes: { iterations: 3, memory_kib: 65536 },
      },
    ],
  },
];

export async function GET() {
  return NextResponse.json(SAMPLE_TRACES);
}
