/**
 * Zod schemas for Nexus's REST admin API (nexus/internal/transport/http/server.go).
 * Field names mirror the `apiTuple` struct's json tags exactly (snake_case).
 */
import { z } from "zod";
import { pagedListSchema, paginatedListSchema } from "../schema";

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

export const tupleListSchema = pagedListSchema(tupleSchema);
export const paginatedTupleListSchema = paginatedListSchema(tupleSchema);
export type PaginatedTuples = z.infer<typeof paginatedTupleListSchema>;

// POST/DELETE /tuples wrap the tuple list in { tuples: [...] } on write, but
// the write handler's own response is a bare array (toAPITuples).
export const writeTuplesResponseSchema = tupleListSchema;

export const deleteTuplesResponseSchema = z.object({
  status: z.string(),
});

// POST /check response: { allowed, reason, trace?: DecisionNode }.
export const caveatResultSchema = z.object({
  caveat_name: z.string(),
  allowed: z.boolean(),
  reason: z.string().optional(),
  context: z.record(z.string(), z.unknown()).optional(),
});
export type CaveatResult = z.infer<typeof caveatResultSchema>;

export type DecisionNode = {
  node_id?: string;
  namespace: string;
  object: string;
  relation: string;
  subject?: {
    namespace?: string;
    object?: string;
    relation?: string;
  };
  allowed: boolean;
  reason?: string;
  caveat?: CaveatResult;
  rewrite_type?: string;
  children?: DecisionNode[];
};

export const decisionNodeSchema: z.ZodType<DecisionNode> = z.lazy(() =>
  z.object({
    node_id: z.string().optional(),
    namespace: z.string(),
    object: z.string(),
    relation: z.string(),
    subject: z
      .object({
        namespace: z.string().optional(),
        object: z.string().optional(),
        relation: z.string().optional(),
      })
      .optional(),
    allowed: z.boolean(),
    reason: z.string().optional(),
    caveat: caveatResultSchema.optional(),
    rewrite_type: z.string().optional(),
    children: z.array(decisionNodeSchema).optional(),
  })
);

export const checkResponseSchema = z.object({
  allowed: z.boolean(),
  reason: z.string(),
  trace: decisionNodeSchema.optional(),
});
export type CheckResponse = z.infer<typeof checkResponseSchema>;

// Tree Expansion (RFC 7662 / Zanzibar Expand)
export type ExpandTreeNode = {
  type: string;
  tuple?: Tuple;
  children?: ExpandTreeNode[];
};

export const expandTreeNodeSchema: z.ZodType<ExpandTreeNode> = z.lazy(() =>
  z.object({
    type: z.string(),
    tuple: tupleSchema.optional(),
    children: z.array(expandTreeNodeSchema).optional(),
  })
);

export const expandResponseSchema = z.object({
  tree: expandTreeNodeSchema.optional(),
});
export type ExpandResponse = z.infer<typeof expandResponseSchema>;

// Reverse Lookups
export const lookupSubjectsResponseSchema = z.object({
  subjects: z.array(
    z.object({
      namespace: z.string(),
      object: z.string(),
      relation: z.string().optional(),
    })
  ),
});
export type LookupSubjectsResponse = z.infer<typeof lookupSubjectsResponseSchema>;

export const lookupResourcesResponseSchema = z.object({
  resources: z.array(z.string()),
});
export type LookupResourcesResponse = z.infer<typeof lookupResourcesResponseSchema>;

// Namespaces & Rewrites
export type RewriteRule = {
  type: string;
  relation?: string;
  tupleset_relation?: string;
  computed_relation?: string;
  children?: RewriteRule[];
};

export const rewriteRuleSchema: z.ZodType<RewriteRule> = z.lazy(() =>
  z.object({
    type: z.string(),
    relation: z.string().optional(),
    tupleset_relation: z.string().optional(),
    computed_relation: z.string().optional(),
    children: z.array(rewriteRuleSchema).optional(),
  })
);

export const relationDefinitionSchema = z.object({
  rewrite: rewriteRuleSchema.optional(),
});

export const namespaceSchema = z.object({
  name: z.string(),
  relations: z.record(z.string(), relationDefinitionSchema).default({}),
  created_at: z.string().optional(),
  updated_at: z.string().optional(),
});
export type NamespaceSchema = z.infer<typeof namespaceSchema>;

export const namespaceListSchema = z.array(namespaceSchema);

// Caveats (ABAC in CEL)
export const caveatSchema = z.object({
  name: z.string(),
  cel_expression: z.string(),
  created_at: z.string().optional(),
});
export type CaveatDefinition = z.infer<typeof caveatSchema>;

export const caveatListSchema = z.array(caveatSchema);

