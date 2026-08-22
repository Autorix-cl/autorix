import { z } from "zod";
import { pagedListSchema, paginatedListSchema } from "../schema";

export const policySchema = z.object({
  ID: z.string(),
  TenantID: z.string().optional().default("default"),
  Name: z.string(),
  Description: z.string().optional().default(""),
  Expression: z.string(),
  Priority: z.number().optional().default(1),
  Enabled: z.boolean().optional().default(true),
  Labels: z.record(z.string(), z.string()).optional().default({}),
  CreatedAt: z.string().optional().default(""),
  UpdatedAt: z.string().optional().default(""),
});

export type Policy = z.infer<typeof policySchema>;

export const policyListSchema = pagedListSchema(policySchema);
export const paginatedPolicyListSchema = paginatedListSchema(policySchema);
export type PaginatedPolicies = z.infer<typeof paginatedPolicyListSchema>;

export const policyResultSchema = z.object({
  PolicyID: z.string().optional().default(""),
  PolicyName: z.string().optional().default(""),
  Passed: z.boolean(),
  Error: z.string().optional(),
  Expression: z.string().optional().default(""),
});

export const evaluateResponseSchema = z.object({
  AllPassed: z.boolean(),
  Results: z.array(policyResultSchema).optional().default([]),
  TotalEvaluated: z.number().optional().default(0),
});

export const deletePolicyResponseSchema = z.object({
  status: z.string(),
});
