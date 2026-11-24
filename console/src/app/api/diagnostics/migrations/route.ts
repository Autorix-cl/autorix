import { NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";
import { requireOperator } from "@/lib/auth/session";
import type { MigrationStatus } from "@/lib/api/schemas/diagnostics";

const MIGRATIONS_TIMEOUT_MS = 5_000;

interface FleetInstanceMigrationRecord {
  id?: string;
  engine_type?: string;
  environment?: string;
  schema_version?: number;
  expected_schema_version?: number;
  last_heartbeat_at?: string;
  updated_at?: string;
  metadata?: {
    schema_version?: number;
    expected_schema_version?: number;
    last_migrated_at?: string;
  };
}

export async function GET() {
  const auth = await requireOperator("diagnostics:read");
  if (!auth.ok) return auth.response;

  try {
    const res = await fetch(`${getServiceUrl("argus")}/v1/instances?limit=200`, {
      cache: "no-store",
      signal: AbortSignal.timeout(MIGRATIONS_TIMEOUT_MS),
    });

    if (!res.ok) {
      return NextResponse.json(
        {
          error: `Argus fleet inventory returned HTTP ${res.status}`,
          source: "unavailable",
          migrations: [],
        },
        { status: 503, headers: { "X-Autorix-Source": "unavailable" } },
      );
    }

    const body = await res.json();
    const instances: FleetInstanceMigrationRecord[] = Array.isArray(body)
      ? body
      : Array.isArray(body?.data)
        ? body.data
        : [];

    const migrations: MigrationStatus[] = instances.map((inst) => {
      const current = Number(inst.schema_version ?? inst.metadata?.schema_version ?? 0);
      const expected = Number(inst.expected_schema_version ?? inst.metadata?.expected_schema_version ?? current);
      const pending = Math.max(0, expected - current);
      return {
        engine_type: inst.engine_type || "unknown",
        instance_id: inst.id || "unknown",
        environment: inst.environment || "default",
        current_schema_version: Number.isFinite(current) ? current : 0,
        expected_schema_version: Number.isFinite(expected) ? expected : 0,
        up_to_date: pending === 0,
        pending_migrations_count: pending,
        last_migrated_at: inst.metadata?.last_migrated_at || inst.updated_at || inst.last_heartbeat_at || new Date(0).toISOString(),
        status: pending === 0 ? "up_to_date" : "behind",
      };
    });

    return NextResponse.json(migrations, {
      status: 200,
      headers: { "X-Autorix-Source": "live" },
    });
  } catch (err) {
    return NextResponse.json(
      {
        error: err instanceof Error ? err.message : "Migration status source unavailable",
        source: "unavailable",
        migrations: [],
      },
      { status: 503, headers: { "X-Autorix-Source": "unavailable" } },
    );
  }
}
