import { z } from "zod";

export const auditOutcomeSchema = z.enum(["success", "denied", "failed"]);
export type AuditOutcome = z.infer<typeof auditOutcomeSchema>;

export const auditEntrySchema = z.object({
  id: z.string(),
  timestamp: z.string(),
  actor: z.string(),
  actor_type: z.string().optional().default("operator"),
  action: z.string(),
  resource_type: z.string(),
  resource_id: z.string(),
  environment: z.string().optional().default("production"),
  request_id: z.string().optional(),
  ip_address: z.string().optional(),
  user_agent: z.string().optional(),
  outcome: z.string(),
  reason: z.string().optional(),
  before_state: z.record(z.string(), z.unknown()).nullable().optional(),
  after_state: z.record(z.string(), z.unknown()).nullable().optional(),
  hash: z.string().optional(),
  prev_hash: z.string().optional(),
  sequence: z.number().optional(),
});

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
