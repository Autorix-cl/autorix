import { z } from "zod";
import { pagedListSchema } from "../schema";

export const policySchema = z.object({
  id: z.string().optional(),
  ID: z.string().optional(),
  tenant_id: z.string().optional(),
  TenantID: z.string().optional(),
  name: z.string().optional(),
  Name: z.string().optional(),
  description: z.string().optional(),
  Description: z.string().optional(),
  expression: z.string().optional(),
  Expression: z.string().optional(),
  priority: z.number().optional(),
  Priority: z.number().optional(),
  enabled: z.boolean().optional(),
  Enabled: z.boolean().optional(),
  labels: z.record(z.string(), z.string()).nullable().optional(),
  Labels: z.record(z.string(), z.string()).nullable().optional(),
  created_at: z.string().optional(),
  CreatedAt: z.string().optional(),
  updated_at: z.string().optional(),
  UpdatedAt: z.string().optional(),
}).transform((data) => ({
  ID: data.id || data.ID || "",
  TenantID: data.tenant_id || data.TenantID || "",
  Name: data.name || data.Name || "",
  Description: data.description || data.Description || "",
  Expression: data.expression || data.Expression || "",
  Priority: data.priority ?? data.Priority ?? 1,
  Enabled: data.enabled ?? data.Enabled ?? true,
  Labels: data.labels || data.Labels || {},
  CreatedAt: data.created_at || data.CreatedAt || "",
  UpdatedAt: data.updated_at || data.UpdatedAt || "",
}));

export type Policy = z.infer<typeof policySchema>;

export const policyListSchema = pagedListSchema(policySchema);

export const policyResultSchema = z.object({
  policy_id: z.string().optional(),
  PolicyID: z.string().optional(),
  policy_name: z.string().optional(),
  PolicyName: z.string().optional(),
  passed: z.boolean().optional(),
  Passed: z.boolean().optional(),
  error: z.string().optional(),
  Error: z.string().optional(),
  expression: z.string().optional(),
  Expression: z.string().optional(),
}).transform((data) => ({
  PolicyID: data.policy_id || data.PolicyID || "",
  PolicyName: data.policy_name || data.PolicyName || "",
  Passed: data.passed ?? data.Passed ?? false,
  Error: data.error || data.Error,
  Expression: data.expression || data.Expression || "",
}));

export const evaluateResponseSchema = z.object({
  all_passed: z.boolean().optional(),
  AllPassed: z.boolean().optional(),
  results: z.array(policyResultSchema).nullable().optional(),
  Results: z.array(policyResultSchema).nullable().optional(),
  total_evaluated: z.number().optional(),
  TotalEvaluated: z.number().optional(),
}).transform((data) => ({
  AllPassed: data.all_passed ?? data.AllPassed ?? false,
  Results: data.results || data.Results || [],
  TotalEvaluated: data.total_evaluated ?? data.TotalEvaluated ?? 0,
}));

export const deletePolicyResponseSchema = z.object({
  status: z.string(),
});
