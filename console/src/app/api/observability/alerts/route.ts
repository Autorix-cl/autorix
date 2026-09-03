import { NextResponse } from "next/server";
import type { AlertEvent } from "@/lib/api/schemas/observability";

const SAMPLE_ALERTS: AlertEvent[] = [
  {
    id: "evt-1",
    rule_id: "rule-err-aegis",
    rule_name: "High Ingress Error Rate",
    engine_type: "aegis",
    severity: "warning",
    state: "firing",
    value: 0.038,
    threshold: 0.02,
    triggered_at: new Date(Date.now() - 1000 * 60 * 5).toISOString(),
  },
  {
    id: "evt-2",
    rule_id: "rule-p99-nexus",
    rule_name: "Elevated ReBAC Check Latency",
    engine_type: "nexus",
    severity: "warning",
    state: "acknowledged",
    value: 28.5,
    threshold: 25.0,
    triggered_at: new Date(Date.now() - 1000 * 60 * 30).toISOString(),
  },
  {
    id: "evt-3",
    rule_id: "rule-hermes-cert",
    rule_name: "IdP Certificate Expiring Soon",
    engine_type: "hermes",
    severity: "critical",
    state: "firing",
    value: 14.0,
    threshold: 15.0,
    triggered_at: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString(),
  },
  {
    id: "evt-4",
    rule_id: "rule-ego-mfa",
    rule_name: "MFA Verification Failures Spiking",
    engine_type: "ego",
    severity: "info",
    state: "resolved",
    value: 0.08,
    threshold: 0.05,
    triggered_at: new Date(Date.now() - 1000 * 60 * 60 * 6).toISOString(),
    resolved_at: new Date(Date.now() - 1000 * 60 * 60 * 5).toISOString(),
  },
];

export async function GET() {
  return NextResponse.json(SAMPLE_ALERTS);
}
