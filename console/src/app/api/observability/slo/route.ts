import { NextResponse } from "next/server";
import type { SLODefinition } from "@/lib/api/schemas/observability";

const SAMPLE_SLOS: SLODefinition[] = [
  {
    id: "slo-nexus-availability",
    name: "Nexus ReBAC Authorization Availability",
    engine_type: "nexus",
    target_percentage: 99.95,
    current_percentage: 99.98,
    error_budget_remaining_percent: 78.4,
    burn_rate: 0.62,
    window: "30d",
  },
  {
    id: "slo-aegis-latency",
    name: "Aegis Ingress Proxy p95 < 5ms",
    engine_type: "aegis",
    target_percentage: 99.0,
    current_percentage: 99.4,
    error_budget_remaining_percent: 86.2,
    burn_rate: 0.45,
    window: "30d",
  },
  {
    id: "slo-themis-evaluation",
    name: "Themis ABAC Policy Latency p95 < 8ms",
    engine_type: "themis",
    target_percentage: 99.5,
    current_percentage: 99.85,
    error_budget_remaining_percent: 92.0,
    burn_rate: 0.31,
    window: "30d",
  },
  {
    id: "slo-ego-auth",
    name: "Ego Session Validation Availability",
    engine_type: "ego",
    target_percentage: 99.9,
    current_percentage: 99.92,
    error_budget_remaining_percent: 64.8,
    burn_rate: 1.05,
    window: "30d",
  },
  {
    id: "slo-vulcan-verify",
    name: "Vulcan Key Cryptographic Verification",
    engine_type: "vulcan",
    target_percentage: 99.99,
    current_percentage: 100.0,
    error_budget_remaining_percent: 100.0,
    burn_rate: 0.0,
    window: "30d",
  },
];

export async function GET() {
  return NextResponse.json(SAMPLE_SLOS);
}
