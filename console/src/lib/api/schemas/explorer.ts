import { z } from "zod";

// 1. Unified Subject Schema (P6-S8-T1)
export const unifiedSubjectSchema = z.object({
  id: z.string(),
  identity: z.object({
    id: z.string(),
    state: z.string(),
    traits: z.record(z.string(), z.unknown()),
    created_at: z.string().optional(),
    schema_id: z.string().optional(),
  }).optional(),
  sessions: z.array(
    z.object({
      id: z.string(),
      identity_id: z.string(),
      active: z.boolean().default(true),
      ip_address: z.string().optional(),
      user_agent: z.string().optional(),
      created_at: z.string().optional(),
      expires_at: z.string().optional(),
    })
  ).default([]),
  relations: z.array(
    z.object({
      namespace: z.string(),
      object: z.string(),
      relation: z.string(),
      subject: z.string(),
      caveat: z.string().optional(),
    })
  ).default([]),
  oauth_grants: z.array(
    z.object({
      id: z.string(),
      client_id: z.string(),
      scope: z.string().optional(),
      created_at: z.string().optional(),
      expires_at: z.string().optional(),
    })
  ).default([]),
  api_keys: z.array(
    z.object({
      id: z.string(),
      name: z.string(),
      prefix: z.string(),
      scopes: z.array(z.string()).default([]),
      call_count: z.number().default(0),
      last_used_at: z.string().optional(),
    })
  ).default([]),
  enterprise_linkage: z.object({
    provider_id: z.string().optional(),
    external_id: z.string().optional(),
    email: z.string().optional(),
    active: z.boolean().default(true),
    last_sync_at: z.string().optional(),
  }).optional(),
});
export type UnifiedSubject = z.infer<typeof unifiedSubjectSchema>;

// 2. Effective Access Explorer Schema (P6-S8-T2)
export const effectiveAccessResultSchema = z.object({
  subject: z.string(),
  resource: z.string(),
  action: z.string(),
  allowed: z.boolean(),
  reason: z.string(),
  engine_breakdown: z.object({
    aegis: z.object({
      matched: z.boolean(),
      rule_id: z.string().optional(),
      rule_name: z.string().optional(),
      upstream: z.string().optional(),
    }),
    nexus: z.object({
      checked: z.boolean(),
      namespace: z.string().optional(),
      object: z.string().optional(),
      relation: z.string().optional(),
      allowed: z.boolean(),
      path: z.string().optional(),
    }),
    themis: z.object({
      evaluated: z.boolean(),
      policies_matched: z.number().default(0),
      allowed: z.boolean(),
      failed_policy: z.string().optional(),
    }),
  }),
});
export type EffectiveAccessResult = z.infer<typeof effectiveAccessResultSchema>;

// 3. End-to-End Request Simulation Trace Schema (P6-S8-T3)
export const simulationTraceStepSchema = z.object({
  step: z.string(),
  engine: z.enum(["aegis", "ego", "janus", "vulcan", "nexus", "themis", "upstream"]),
  status: z.enum(["pass", "fail", "skip"]),
  latency_ms: z.number(),
  details: z.record(z.string(), z.unknown()).default({}),
});
export type SimulationTraceStep = z.infer<typeof simulationTraceStepSchema>;

export const requestSimulationTraceSchema = z.object({
  trace_id: z.string(),
  request: z.object({
    method: z.string(),
    path: z.string(),
    headers: z.record(z.string(), z.string()).default({}),
    subject: z.string(),
  }),
  steps: z.array(simulationTraceStepSchema),
  outcome: z.object({
    allowed: z.boolean(),
    status_code: z.number(),
    final_decision: z.string(),
    latency_total_ms: z.number(),
  }),
});
export type RequestSimulationTrace = z.infer<typeof requestSimulationTraceSchema>;

// 4. Configuration Consistency Finding Schema (P6-S8-T4)
export const consistencyFindingSchema = z.object({
  id: z.string(),
  severity: z.enum(["critical", "warning", "info"]),
  category: z.enum(["nexus", "aegis", "oauth2", "vulcan", "hermes", "themis", "general"]),
  title: z.string(),
  description: z.string(),
  remediation: z.string(),
  remediation_link: z.string(),
});
export const consistencyFindingListSchema = z.array(consistencyFindingSchema);
export type ConsistencyFinding = z.infer<typeof consistencyFindingSchema>;
