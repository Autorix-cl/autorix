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
});

export const ruleSchema = z.object({
  id: z.string(),
  description: z.string().optional(),
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

// POST /rules/test-match response: { matched: boolean, rule?: Rule }.
export const testMatchResponseSchema = z.object({
  matched: z.boolean(),
  rule: ruleSchema.optional(),
});
