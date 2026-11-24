import { NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";
import { buildInternalAuthHeaders } from "@/lib/api/internal-headers";
import { requireOperator } from "@/lib/auth/session";
import type { ConfigDriftFinding } from "@/lib/api/schemas/diagnostics";

const DRIFT_TIMEOUT_MS = 5_000;

interface FleetInstanceRecord {
  id?: string;
  engine_type?: string;
  environment?: string;
  version?: string;
  config_hash?: string;
}

export async function GET() {
  const auth = await requireOperator("diagnostics:read");
  if (!auth.ok) return auth.response;

  try {
    const res = await fetch(`${getServiceUrl("argus")}/v1/instances?limit=200`, {
      cache: "no-store",
      headers: buildInternalAuthHeaders(),
      signal: AbortSignal.timeout(DRIFT_TIMEOUT_MS),
    });

    if (!res.ok) {
      return NextResponse.json(
        {
          error: `Argus fleet inventory returned HTTP ${res.status}`,
          source: "unavailable",
          findings: [],
        },
        { status: 503, headers: { "X-Autorix-Source": "unavailable" } },
      );
    }

    const body = await res.json();
    const instances: FleetInstanceRecord[] = Array.isArray(body) ? body : Array.isArray(body?.data) ? body.data : [];

    // Group live instances by engine_type + environment and detect real version/config divergence
    const groups = new Map<string, FleetInstanceRecord[]>();
    for (const inst of instances) {
      const engineType = inst.engine_type || "unknown";
      const env = inst.environment || "default";
      const key = `${engineType}::${env}`;
      const existing = groups.get(key) || [];
      existing.push(inst);
      groups.set(key, existing);
    }

    const findings: ConfigDriftFinding[] = [];
    for (const [key, group] of groups.entries()) {
      if (group.length < 2) continue;
      const [engineType, environment] = key.split("::");

      const versions = group.map((i) => ({ id: i.id || "unknown", val: i.version || "" })).filter((v) => v.val);
      if (versions.length >= 2) {
        const referenceVersion = versions[0].val;
        const divergent = versions
          .filter((v) => v.val !== referenceVersion)
          .map((v) => ({ instance_id: v.id, value: v.val }));
        if (divergent.length > 0) {
          findings.push({
            id: `drift-${engineType}-${environment}-version`,
            engine_type: engineType,
            environment,
            parameter: "engine.version",
            reference_value: referenceVersion,
            divergent_instances: divergent,
            severity: "warning",
          });
        }
      }

      const hashes = group.map((i) => ({ id: i.id || "unknown", val: i.config_hash || "" })).filter((v) => v.val);
      if (hashes.length >= 2) {
        const referenceHash = hashes[0].val;
        const divergent = hashes
          .filter((h) => h.val !== referenceHash)
          .map((h) => ({ instance_id: h.id, value: h.val }));
        if (divergent.length > 0) {
          findings.push({
            id: `drift-${engineType}-${environment}-config-hash`,
            engine_type: engineType,
            environment,
            parameter: "config.checksum",
            reference_value: referenceHash,
            divergent_instances: divergent,
            severity: "critical",
          });
        }
      }
    }

    return NextResponse.json(findings, {
      status: 200,
      headers: { "X-Autorix-Source": "live" },
    });
  } catch (err) {
    return NextResponse.json(
      {
        error: err instanceof Error ? err.message : "Fleet configuration drift source unavailable",
        source: "unavailable",
        findings: [],
      },
      { status: 503, headers: { "X-Autorix-Source": "unavailable" } },
    );
  }
}
