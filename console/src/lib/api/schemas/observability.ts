import { z } from "zod";

// 1. Fleet & Engine Metrics (P7-S1-T6, P7-S3-T2, P7-S3-T3)
export const engineMetricSummarySchema = z.object({
  engine_type: z.string(),
  status: z.string(),
  instance_count: z.number(),
  requests_total: z.number(),
  requests_per_second: z.number(),
  error_rate: z.number().nullable(),
  latency_p50_ms: z.number().nullable(),
  latency_p95_ms: z.number().nullable(),
  latency_p99_ms: z.number().nullable(),
  auth_decisions_total: z.number().nullable(),
  auth_allow_rate: z.number().nullable(),
});
export type EngineMetricSummary = z.infer<typeof engineMetricSummarySchema>;

export const fleetMetricsSummarySchema = z.object({
  timestamp: z.string(),
  total_engines: z.number(),
  total_instances: z.number(),
  healthy_instances: z.number(),
  requests_total: z.number(),
  fleet_qps: z.number(),
  fleet_error_rate: z.number(),
  fleet_latency_p95_ms: z.number().nullable(),
  auth_decisions_total: z.number().nullable(),
  auth_allow_rate: z.number().nullable(),
  engines: z.array(engineMetricSummarySchema),
});
export type FleetMetricsSummary = z.infer<typeof fleetMetricsSummarySchema>;

// 2. Structured Log Entry (P7-S2-T4)
export const logEntrySchema = z.object({
  id: z.string(),
  timestamp: z.string(),
  engine: z.string(),
  instance_id: z.string().optional(),
  level: z.enum(["debug", "info", "warn", "error"]),
  message: z.string(),
  request_id: z.string().optional(),
  correlation_id: z.string().optional(),
  trace_id: z.string().optional(),
  attributes: z.record(z.string(), z.unknown()).default({}),
});
export const logEntryListSchema = z.array(logEntrySchema);
export type LogEntry = z.infer<typeof logEntrySchema>;

// 3. Distributed Tracing & Span Waterfall (P7-S2-T5)
export const traceSpanSchema = z.object({
  span_id: z.string(),
  parent_span_id: z.string().optional(),
  trace_id: z.string(),
  engine: z.string(),
  operation: z.string(),
  status: z.enum(["ok", "error"]),
  start_time_ms: z.number(),
  duration_ms: z.number(),
  attributes: z.record(z.string(), z.unknown()).default({}),
});
export type TraceSpan = z.infer<typeof traceSpanSchema>;

export const traceDetailSchema = z.object({
  trace_id: z.string(),
  root_operation: z.string(),
  root_engine: z.string(),
  total_duration_ms: z.number(),
  timestamp: z.string(),
  status: z.enum(["ok", "error"]),
  spans: z.array(traceSpanSchema),
});
export const traceListSchema = z.array(traceDetailSchema);
export type TraceDetail = z.infer<typeof traceDetailSchema>;

// 4. Alert Rules & Events (P7-S3-T4, P7-S3-T6)
export const alertRuleSchema = z.object({
  id: z.string(),
  name: z.string(),
  engine_type: z.string(),
  severity: z.enum(["critical", "warning", "info"]),
  metric: z.string(),
  threshold: z.number().nullable(),
  operator: z.enum(["gt", "lt", "gte", "lte"]).nullable(),
  duration: z.string(),
  enabled: z.boolean().default(true),
});
export const alertRuleListSchema = z.array(alertRuleSchema);
export type AlertRule = z.infer<typeof alertRuleSchema>;

export const alertEventSchema = z.object({
  id: z.string(),
  rule_id: z.string(),
  rule_name: z.string(),
  engine_type: z.string(),
  severity: z.enum(["critical", "warning", "info"]),
  state: z.enum(["firing", "acknowledged", "silenced", "resolved"]),
  value: z.number(),
  threshold: z.number().nullable(),
  triggered_at: z.string(),
  resolved_at: z.string().optional(),
});
export const alertEventListSchema = z.array(alertEventSchema);
export type AlertEvent = z.infer<typeof alertEventSchema>;

// 5. SLO & Error Budgets (P7-S3-T7)
export const sloDefinitionSchema = z.object({
  id: z.string(),
  name: z.string(),
  engine_type: z.string(),
  target_percentage: z.number(),
  current_percentage: z.number(),
  error_budget_remaining_percent: z.number(),
  burn_rate: z.number(),
  window: z.string(),
});
export const sloListSchema = z.array(sloDefinitionSchema);
export type SLODefinition = z.infer<typeof sloDefinitionSchema>;
