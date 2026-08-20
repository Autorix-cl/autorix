import { z } from "zod";

export const auditOutcomeSchema = z.enum(["success", "denied", "failed"]);
export type AuditOutcome = z.infer<typeof auditOutcomeSchema>;

export const auditEntrySchema = z.object({
  id: z.string(),
  created_at: z.string().optional(),
  timestamp: z.string().optional(),
  actor: z.string().optional(),
  actor_id: z.string().optional(),
  actor_type: z.string().optional().default("operator"),
  action: z.string(),
  resource_type: z.string(),
  resource_id: z.string(),
  environment: z.string().optional().default("production"),
  request_id: z.string().optional().default(""),
  ip_address: z.string().optional(),
  source_ip: z.string().optional(),
  user_agent: z.string().optional().default(""),
  outcome: z.string().optional().default("success"),
  reason: z.string().optional(),
  before_state: z.record(z.string(), z.unknown()).nullable().optional(),
  after_state: z.record(z.string(), z.unknown()).nullable().optional(),
  hash: z.string().optional(),
  record_hash: z.string().optional(),
  prev_hash: z.string().optional().default(""),
  sequence: z.number().optional().default(0),
}).transform((data) => ({
  id: data.id,
  timestamp: data.created_at || data.timestamp || new Date().toISOString(),
  actor: data.actor || data.actor_id || "system",
  actor_type: data.actor_type || "operator",
  action: data.action,
  resource_type: data.resource_type,
  resource_id: data.resource_id,
  environment: data.environment || "production",
  request_id: data.request_id || "",
  ip_address: data.source_ip || data.ip_address || "127.0.0.1",
  user_agent: data.user_agent || "",
  outcome: data.outcome || "success",
  reason: data.reason,
  before_state: data.before_state,
  after_state: data.after_state,
  hash: data.record_hash || data.hash || "",
  prev_hash: data.prev_hash || "",
  sequence: data.sequence ?? 0,
}));

export type AuditEntry = z.infer<typeof auditEntrySchema>;

export const auditListResponseSchema = z.union([
  z.array(auditEntrySchema),
  z.object({
    data: z.array(auditEntrySchema),
    next_cursor: z.string().optional(),
    has_more: z.boolean().optional(),
    total_count: z.number().optional(),
  }),
]);

export type AuditListResponse = z.infer<typeof auditListResponseSchema>;

export const auditVerificationSchema = z.object({
  verified: z.boolean(),
  chain_length: z.number(),
  head_hash: z.string(),
  genesis_hash: z.string().optional(),
  broken_link: z
    .object({
      id: z.string(),
      sequence: z.number(),
      expected_prev_hash: z.string(),
      actual_prev_hash: z.string(),
    })
    .nullable()
    .optional(),
  verified_at: z.string(),
  algorithm: z.string().optional().default("SHA-256"),
});

export type AuditVerification = z.infer<typeof auditVerificationSchema>;

export const auditExportResponseSchema = z.object({
  format: z.string(),
  content: z.string().optional(),
  download_url: z.string().optional(),
  filename: z.string().optional(),
  record_count: z.number().optional(),
  generated_at: z.string().optional(),
  signature: z.string().optional(),
});

export type AuditExportResponse = z.infer<typeof auditExportResponseSchema>;
