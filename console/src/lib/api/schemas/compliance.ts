import { z } from "zod";

export const complianceControlStatusSchema = z.enum(["compliant", "needs_review", "non_compliant"]);
export type ComplianceControlStatus = z.infer<typeof complianceControlStatusSchema>;

export const complianceEvidenceItemSchema = z.object({
  id: z.string(),
  framework: z.string(),
  control_id: z.string(),
  control_name: z.string(),
  status: complianceControlStatusSchema,
  evidence_type: z.string(),
  description: z.string(),
  engine: z.string().optional(),
  last_evaluated_at: z.string(),
  evaluator: z.string(),
  artifacts_count: z.number().optional().default(1),
  details: z.record(z.string(), z.unknown()).optional(),
});

export type ComplianceEvidenceItem = z.infer<typeof complianceEvidenceItemSchema>;

export const complianceEvidenceResponseSchema = z.union([
  z.array(complianceEvidenceItemSchema),
  z.object({
    data: z.array(complianceEvidenceItemSchema),
    summary: z
      .object({
        total_controls: z.number().optional(),
        compliant_controls: z.number().optional(),
        review_required: z.number().optional(),
        score_percent: z.number().optional(),
      })
      .optional(),
  }),
]);

export type ComplianceEvidenceResponse = z.infer<typeof complianceEvidenceResponseSchema>;

export const compliancePackageRequestSchema = z.object({
  framework: z.string(),
  period: z.string().optional(),
  from: z.string().optional(),
  to: z.string().optional(),
  include_audit_trail: z.boolean().optional().default(true),
  include_access_review: z.boolean().optional().default(true),
  include_cryptographic_proof: z.boolean().optional().default(true),
});

export type CompliancePackageRequest = z.infer<typeof compliancePackageRequestSchema>;

export const compliancePackageResponseSchema = z.object({
  package_id: z.string(),
  framework: z.string(),
  period_start: z.string(),
  period_end: z.string(),
  status: z.string(),
  generated_at: z.string(),
  download_url: z.string().optional(),
  content: z.string().optional(),
  summary: z
    .object({
      total_controls: z.number(),
      passing_controls: z.number(),
      failing_controls: z.number(),
      review_required: z.number(),
    })
    .optional(),
  controls: z.array(complianceEvidenceItemSchema).optional(),
});

export type CompliancePackageResponse = z.infer<typeof compliancePackageResponseSchema>;
