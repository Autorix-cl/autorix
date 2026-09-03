import { NextRequest, NextResponse } from "next/server";
import type { AlertRule } from "@/lib/api/schemas/observability";

const SAMPLE_RULES: AlertRule[] = [
  {
    id: "rule-err-aegis",
    name: "High Ingress Error Rate",
    engine_type: "aegis",
    severity: "warning",
    metric: "http_requests_error_ratio",
    threshold: 0.02,
    operator: "gt",
    duration: "2m",
    enabled: true,
  },
  {
    id: "rule-p99-nexus",
    name: "Elevated ReBAC Check Latency",
    engine_type: "nexus",
    severity: "warning",
    metric: "nexus_check_duration_p99_ms",
    threshold: 25.0,
    operator: "gt",
    duration: "5m",
    enabled: true,
  },
  {
    id: "rule-hermes-cert",
    name: "IdP Certificate Expiring Soon",
    engine_type: "hermes",
    severity: "critical",
    metric: "hermes_idp_cert_days_remaining",
    threshold: 15.0,
    operator: "lt",
    duration: "1h",
    enabled: true,
  },
  {
    id: "rule-themis-err",
    name: "ABAC Policy Compile Failures",
    engine_type: "themis",
    severity: "critical",
    metric: "themis_policy_compile_errors_total",
    threshold: 1.0,
    operator: "gte",
    duration: "1m",
    enabled: true,
  },
  {
    id: "rule-vulcan-key",
    name: "Excessive Invalid Key Probing",
    engine_type: "vulcan",
    severity: "warning",
    metric: "vulcan_verify_failures_per_sec",
    threshold: 50.0,
    operator: "gt",
    duration: "3m",
    enabled: true,
  },
];

export async function GET() {
  return NextResponse.json(SAMPLE_RULES);
}

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const newRule: AlertRule = {
      id: `rule-${Date.now()}`,
      name: body.name || "Custom Alert Rule",
      engine_type: body.engine_type || "aegis",
      severity: body.severity || "warning",
      metric: body.metric || "custom_metric",
      threshold: Number(body.threshold) || 10,
      operator: body.operator || "gt",
      duration: body.duration || "5m",
      enabled: true,
    };
    return NextResponse.json(newRule, { status: 201 });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to create rule";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
