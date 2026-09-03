import { NextResponse } from "next/server";
import type { MigrationStatus } from "@/lib/api/schemas/diagnostics";

const MIGRATION_STATES: MigrationStatus[] = [
  {
    engine_type: "argus",
    instance_id: "argus-inst-1",
    environment: "production",
    current_schema_version: 5,
    expected_schema_version: 5,
    up_to_date: true,
    pending_migrations_count: 0,
    last_migrated_at: "2026-09-01T14:30:00Z",
  },
  {
    engine_type: "ego",
    instance_id: "ego-inst-1",
    environment: "production",
    current_schema_version: 6,
    expected_schema_version: 6,
    up_to_date: true,
    pending_migrations_count: 0,
    last_migrated_at: "2026-09-02T10:15:00Z",
  },
  {
    engine_type: "nexus",
    instance_id: "nexus-inst-1",
    environment: "production",
    current_schema_version: 4,
    expected_schema_version: 4,
    up_to_date: true,
    pending_migrations_count: 0,
    last_migrated_at: "2026-09-02T11:45:00Z",
  },
  {
    engine_type: "themis",
    instance_id: "themis-inst-1",
    environment: "production",
    current_schema_version: 3,
    expected_schema_version: 3,
    up_to_date: true,
    pending_migrations_count: 0,
    last_migrated_at: "2026-09-02T15:20:00Z",
  },
  {
    engine_type: "aegis",
    instance_id: "aegis-inst-1",
    environment: "production",
    current_schema_version: 4,
    expected_schema_version: 4,
    up_to_date: true,
    pending_migrations_count: 0,
    last_migrated_at: "2026-09-02T16:00:00Z",
  },
  {
    engine_type: "vulcan",
    instance_id: "vulcan-inst-1",
    environment: "production",
    current_schema_version: 3,
    expected_schema_version: 3,
    up_to_date: true,
    pending_migrations_count: 0,
    last_migrated_at: "2026-09-02T18:00:00Z",
  },
  {
    engine_type: "hermes",
    instance_id: "hermes-inst-1",
    environment: "production",
    current_schema_version: 3,
    expected_schema_version: 3,
    up_to_date: true,
    pending_migrations_count: 0,
    last_migrated_at: "2026-09-02T19:30:00Z",
  },
];

export async function GET() {
  return NextResponse.json(MIGRATION_STATES);
}
