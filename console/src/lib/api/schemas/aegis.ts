/**
 * Zod schemas for Aegis's REST admin API (aegis/internal/transport/http/server.go).
 * Field names mirror the `core.Rule` struct's json tags exactly (aegis/internal/core).
 */
import { z } from "zod";
import { pagedListSchema } from "../schema";

export const matchConfigSchema = z.object({
  url: z.string(),
  methods: z.array(z.string()),
});

export const handlerConfigSchema = z.object({
  handler: z.string(),
  config: z.record(z.string(), z.unknown()).optional(),
});

export const upstreamConfigSchema = z.object({
  url: z.string(),
  strip_prefix: z.string().optional(),
  rewrite: z.string().optional(),
});

export const ruleSchema = z.object({
  id: z.string(),
  description: z.string().optional(),
  order_idx: z.number().optional(),
  match: matchConfigSchema,
  authenticators: z.array(handlerConfigSchema),
  authorizer: handlerConfigSchema,
  mutators: z.array(handlerConfigSchema),
  upstream: upstreamConfigSchema,
});
export type Rule = z.infer<typeof ruleSchema>;

export const ruleListSchema = pagedListSchema(ruleSchema);

export const deleteRuleResponseSchema = z.object({
  status: z.string(),
});

export const sessionSchema = z.object({
  subject: z.string().default(""),
  scopes: z.array(z.string()).default([]),
  extra: z.record(z.string(), z.unknown()).default({}),
  headers: z.record(z.string(), z.union([z.string(), z.array(z.string())])).default({}),
});
export type Session = z.infer<typeof sessionSchema>;

export const pipelineTraceStepSchema = z.object({
  stage: z.string(), // "match", "authenticator", "authorizer", "mutator", "upstream"
  handler: z.string().optional(),
  status: z.string(), // "success", "failure", "skipped"
  details: z.string().optional(),
  allowed: z.boolean().optional(),
  session: sessionSchema.optional(),
  mutated_headers: z.record(z.string(), z.union([z.string(), z.array(z.string())])).optional(),
  target_url: z.string().optional(),
  error: z.string().optional(),
});
export type PipelineTraceStep = z.infer<typeof pipelineTraceStepSchema>;

export const pipelineTraceSchema = z.object({
  matched_rule_id: z.string().optional(),
  steps: z.array(pipelineTraceStepSchema),
  final_verdict: z.string(), // "allow", "deny", "unauthorized", "error"
  error: z.string().optional(),
});
export type PipelineTrace = z.infer<typeof pipelineTraceSchema>;

// POST /rules/test-match response: { matched: boolean, rule?: Rule, trace?: PipelineTrace }.
export const testMatchResponseSchema = z.object({
  matched: z.boolean(),
  rule: ruleSchema.optional(),
  trace: pipelineTraceSchema.optional(),
});

export const handlerInfoSchema = z.object({
  name: z.string(),
  description: z.string(),
  config_schema: z.record(z.string(), z.unknown()).default({}),
});
export type HandlerInfo = z.infer<typeof handlerInfoSchema>;

export const handlerCatalogueSchema = z.object({
  authenticators: z.array(handlerInfoSchema).default([]),
  authorizers: z.array(handlerInfoSchema).default([]),
  mutators: z.array(handlerInfoSchema).default([]),
});
export type HandlerCatalogue = z.infer<typeof handlerCatalogueSchema>;

export const ruleVersionSchema = z.object({
  version: z.number(),
  description: z.string().optional(),
  rules: z.array(ruleSchema),
  created_at: z.string(),
});
export type RuleVersion = z.infer<typeof ruleVersionSchema>;

export const ruleVersionListSchema = z.array(ruleVersionSchema);

export const reorderRulesRequestSchema = z.object({
  ids: z.array(z.string()),
});
export type ReorderRulesRequest = z.infer<typeof reorderRulesRequestSchema>;

export const reorderResponseSchema = z.object({
  status: z.string(),
});

export const rollbackResponseSchema = z.object({
  status: z.string(),
  version: z.number().optional(),
});

