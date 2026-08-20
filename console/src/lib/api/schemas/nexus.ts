/**
 * Zod schemas for Nexus's REST admin API (nexus/internal/transport/http/server.go).
 * Field names mirror the `apiTuple` struct's json tags exactly (snake_case).
 */
import { z } from "zod";

export const tupleSchema = z.object({
  namespace: z.string(),
  object: z.string(),
  relation: z.string(),
  subject_namespace: z.string(),
  subject_id: z.string(),
  subject_relation: z.string().optional(),
  caveat_name: z.string().optional(),
  caveat_context: z.record(z.string(), z.unknown()).optional(),
});
export type Tuple = z.infer<typeof tupleSchema>;

export const tupleListSchema = z.array(tupleSchema);

// POST/DELETE /tuples wrap the tuple list in { tuples: [...] } on write, but
// the write handler's own response is a bare array (toAPITuples).
export const writeTuplesResponseSchema = tupleListSchema;

export const deleteTuplesResponseSchema = z.object({
  status: z.string(),
});

// POST /check response: { allowed, reason }.
export const checkResponseSchema = z.object({
  allowed: z.boolean(),
  reason: z.string(),
});
