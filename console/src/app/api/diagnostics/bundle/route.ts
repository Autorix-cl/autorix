import { NextResponse } from "next/server";
import type { DiagnosticBundle } from "@/lib/api/schemas/diagnostics";

export async function POST() {
  const bundleId = `diag_${new Date().toISOString().replace(/[:.]/g, "").slice(0, 15)}`;
  const bundle: DiagnosticBundle = {
    bundle_id: bundleId,
    created_at: new Date().toISOString(),
    fleet_summary: {
      total_engines: 7,
      healthy_engines: 7,
      total_instances: 14,
    },
    engines_included: ["aegis", "nexus", "themis", "ego", "janus", "vulcan", "hermes"],
    events_count: 85,
    logs_count: 500,
    redacted: true,
    download_url: `/api/diagnostics/bundle/${bundleId}.tar.gz`,
  };

  return NextResponse.json(bundle, { status: 201 });
}
