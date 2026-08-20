/**
 * Zod schemas for Themis's REST admin API (themis/internal/transport/http/server.go).
 *
 * IMPORTANT: unlike nexus/aegis, themis's core.Policy / EvaluateResponse /
 * PolicyResult structs (themis/internal/core/engine.go) carry NO `json:"..."`
 * tags, so Go's default JSON encoding emits the exact Go field names
 * (PascalCase) instead of snake_case — e.g. "TenantID", "CreatedAt",
 * "AllPassed", "PolicyID". This is a real deviation from the rest of the
 * console's API surface, confirmed by reading the struct definitions.
 */
import { z } from "zod";

export const policySchema = z.object({
  ID: z.string(),
  TenantID: z.string(),
  Name: z.string(),
  Description: z.string().optional(),
  Expression: z.string(),
  Priority: z.number(),
  Enabled: z.boolean(),
  Labels: z.record(z.string(), z.string()).nullable().optional(),
  CreatedAt: z.string(),
  UpdatedAt: z.string(),
});
export type Policy = z.infer<typeof policySchema>;

export const policyListSchema = z.array(policySchema);

export const policyResultSchema = z.object({
  PolicyID: z.string(),
  PolicyName: z.string(),
  Passed: z.boolean(),
  Error: z.string().optional(),
  Expression: z.string(),
});

export const evaluateResponseSchema = z.object({
  AllPassed: z.boolean(),
  Results: z.array(policyResultSchema).nullable(),
  TotalEvaluated: z.number(),
});

export const deletePolicyResponseSchema = z.object({
  status: z.string(),
});
