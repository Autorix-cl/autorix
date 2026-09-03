import { z } from "zod";

// 1. Connectivity Troubleshooter (P7-S4-T1)
export const connectivityStageStatusSchema = z.enum(["healthy", "degraded", "failed", "skipped"]);

export const connectivityProbeResultSchema = z.object({
  probe_id: z.string(),
  timestamp: z.string(),
  source_instance_id: z.string(),
  source_engine: z.string(),
  target_instance_id: z.string(),
  target_engine: z.string(),
  target_endpoint: z.string(),
  dns_status: connectivityStageStatusSchema,
  dns_latency_ms: z.number(),
  tcp_status: connectivityStageStatusSchema,
  tcp_latency_ms: z.number(),
  tls_status: connectivityStageStatusSchema,
  tls_latency_ms: z.number(),
  http_status: connectivityStageStatusSchema,
  http_code: z.number(),
  http_latency_ms: z.number(),
  overall_status: z.enum(["healthy", "degraded", "failed"]),
  error_message: z.string().optional(),
});
export type ConnectivityProbeResult = z.infer<typeof connectivityProbeResultSchema>;

// 2. Configuration Drift Detection (P7-S4-T2)
export const divergentInstanceSchema = z.object({
  instance_id: z.string(),
  value: z.string(),
});

export const configDriftFindingSchema = z.object({
  id: z.string(),
  engine_type: z.string(),
  environment: z.string(),
  parameter: z.string(),
  reference_value: z.string(),
  divergent_instances: z.array(divergentInstanceSchema),
  severity: z.enum(["warning", "critical"]),
});
export const configDriftListSchema = z.array(configDriftFindingSchema);
export type ConfigDriftFinding = z.infer<typeof configDriftFindingSchema>;

// 3. Migration Status View (P7-S4-T3)
export const migrationStatusSchema = z.object({
  engine_type: z.string(),
  instance_id: z.string(),
  environment: z.string(),
  current_schema_version: z.number(),
  expected_schema_version: z.number(),
  up_to_date: z.boolean(),
  pending_migrations_count: z.number(),
  last_migrated_at: z.string(),
});
export const migrationStatusListSchema = z.array(migrationStatusSchema);
export type MigrationStatus = z.infer<typeof migrationStatusSchema>;

// 4. Change Correlation Timeline (P7-S4-T4)
export const changeTimelineEventSchema = z.object({
  id: z.string(),
  timestamp: z.string(),
  event_type: z.enum(["deployment", "config_change", "registration", "schema_migration"]),
  engine_type: z.string(),
  title: z.string(),
  description: z.string(),
  actor: z.string(),
  correlated_error_spike: z.boolean().default(false),
  error_spike_rate: z.number().optional(),
});
export const changeTimelineListSchema = z.array(changeTimelineEventSchema);
export type ChangeTimelineEvent = z.infer<typeof changeTimelineEventSchema>;

// 5. Diagnostic Bundle Export (P7-S4-T5)
export const diagnosticBundleSchema = z.object({
  bundle_id: z.string(),
  created_at: z.string(),
  fleet_summary: z.object({
    total_engines: z.number(),
    healthy_engines: z.number(),
    total_instances: z.number(),
  }),
  engines_included: z.array(z.string()),
  events_count: z.number(),
  logs_count: z.number(),
  redacted: z.boolean(),
  download_url: z.string(),
});
export type DiagnosticBundle = z.infer<typeof diagnosticBundleSchema>;
