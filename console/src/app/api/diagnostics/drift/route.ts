import { NextResponse } from "next/server";
import type { ConfigDriftFinding } from "@/lib/api/schemas/diagnostics";

const SAMPLE_DRIFT_FINDINGS: ConfigDriftFinding[] = [
  {
    id: "drift-aegis-timeout",
    engine_type: "aegis",
    environment: "production",
    parameter: "proxy.upstream_timeout_ms",
    reference_value: "5000",
    divergent_instances: [
      { instance_id: "aegis-inst-2", value: "3000" },
    ],
    severity: "warning",
  },
  {
    id: "drift-nexus-cache",
    engine_type: "nexus",
    environment: "production",
    parameter: "cache.tuple_ttl_seconds",
    reference_value: "300",
    divergent_instances: [
      { instance_id: "nexus-inst-2", value: "60" },
    ],
    severity: "warning",
  },
];

export async function GET() {
  return NextResponse.json(SAMPLE_DRIFT_FINDINGS);
}
