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

// Go engine representation (snake_case)
export const goPolicySchema = z.object({
  id: z.string(),
  tenant_id: z.string().optional().default("default"),
  name: z.string(),
  description: z.string().optional().default(""),
  expression: z.string(),
  priority: z.number().optional().default(1),
  enabled: z.boolean().optional().default(true),
  labels: z.record(z.string(), z.string()).optional().default({}),
  created_at: z.string().optional().default(""),
  updated_at: z.string().optional().default(""),
});
export const goPolicyListSchema = z.array(goPolicySchema);
export type GoPolicy = z.infer<typeof goPolicySchema>;

// Policy Version Snapshot
export const policyVersionSchema = z.object({
  id: z.string(),
  policy_id: z.string(),
  tenant_id: z.string().optional().default("default"),
  version: z.number(),
  name: z.string(),
  description: z.string().optional().default(""),
  expression: z.string(),
  priority: z.number().optional().default(1),
  enabled: z.boolean().optional().default(true),
  labels: z.record(z.string(), z.string()).optional().default({}),
  created_at: z.string().optional().default(""),
});
export const policyVersionListSchema = z.array(policyVersionSchema);
export type PolicyVersion = z.infer<typeof policyVersionSchema>;

// Policy Test Fixture
export const policyFixtureSchema = z.object({
  id: z.string(),
  policy_id: z.string(),
  tenant_id: z.string().optional().default("default"),
  name: z.string(),
  description: z.string().optional().default(""),
  payload: z.record(z.string(), z.unknown()).optional().default({}),
  expected_result: z.boolean(),
  created_at: z.string().optional().default(""),
  updated_at: z.string().optional().default(""),
});
export const policyFixtureListSchema = z.array(policyFixtureSchema);
export type PolicyFixture = z.infer<typeof policyFixtureSchema>;

// Test Suite Run Result
export const fixtureRunResultSchema = z.object({
  fixture_id: z.string(),
  fixture_name: z.string(),
  expected_result: z.boolean(),
  actual_result: z.boolean(),
  passed: z.boolean(),
  error: z.string().optional().default(""),
});

export const testSuiteResultSchema = z.object({
  policy_id: z.string(),
  all_passed: z.boolean(),
  total_tests: z.number(),
  passed_tests: z.number(),
  failed_tests: z.number(),
  results: z.array(fixtureRunResultSchema),
});
export type TestSuiteResult = z.infer<typeof testSuiteResultSchema>;

// CEL Syntax Validation Result
export const validationErrorSchema = z.object({
  line: z.number().optional().default(0),
  column: z.number().optional().default(0),
  message: z.string(),
});

export const validationResultSchema = z.object({
  valid: z.boolean(),
  variables: z.array(z.string()).optional().default([]),
  errors: z.array(validationErrorSchema).optional().default([]),
});
export type ValidationResult = z.infer<typeof validationResultSchema>;

// Dry Run Result
export const dryRunResultSchema = z.object({
  passed: z.boolean(),
  error: z.string().optional().default(""),
});
export type DryRunResult = z.infer<typeof dryRunResultSchema>;

