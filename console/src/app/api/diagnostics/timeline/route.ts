import { NextResponse } from "next/server";
import type { ChangeTimelineEvent } from "@/lib/api/schemas/diagnostics";

const SAMPLE_TIMELINE: ChangeTimelineEvent[] = [
  {
    id: "evt-change-1",
    timestamp: new Date(Date.now() - 1000 * 60 * 12).toISOString(),
    event_type: "config_change",
    engine_type: "aegis",
    title: "Aegis Ingress Rate Limit Reduced",
    description: "Operator adjusted burst quota from 500 to 100 req/s on /api/v1/projects",
    actor: "operator:sec_lead",
    correlated_error_spike: true,
    error_spike_rate: 0.038,
  },
  {
    id: "evt-change-2",
    timestamp: new Date(Date.now() - 1000 * 60 * 45).toISOString(),
    event_type: "deployment",
    engine_type: "nexus",
    title: "Nexus ReBAC Engine v1.4.2 Deployed",
    description: "Rolling update completed across 2 instances with optimized Zanzibar cache",
    actor: "ci:argusd-deployer",
    correlated_error_spike: false,
  },
  {
    id: "evt-change-3",
    timestamp: new Date(Date.now() - 1000 * 60 * 90).toISOString(),
    event_type: "schema_migration",
    engine_type: "themis",
    title: "Themis Database Migration v3 Applied",
    description: "Added partial index on policy version lookup table",
    actor: "system:argus-migrator",
    correlated_error_spike: false,
  },
  {
    id: "evt-change-4",
    timestamp: new Date(Date.now() - 1000 * 60 * 180).toISOString(),
    event_type: "registration",
    engine_type: "vulcan",
    title: "New Vulcan Replicas Enrolled",
    description: "Instance vulcan-inst-2 joined production cluster",
    actor: "system:fleet-orchestrator",
    correlated_error_spike: false,
  },
];

export async function GET() {
  return NextResponse.json(SAMPLE_TIMELINE);
}
