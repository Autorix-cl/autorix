/**
 * Zod schemas for Vulcan's API Key / Macaroon REST API
 * (vulcan/internal/transport/http/server.go, structs in vulcan/internal/core).
 * Field names mirror the Go json tags exactly (snake_case). Pointer fields
 * with `omitempty` (expires_at, last_used_at) are omitted entirely by Go's
 * encoder when nil, so they're modeled as `.optional()`, not `.nullable()`.
 */
import { z } from "zod";
import { pagedListSchema, paginatedListSchema } from "../schema";

// core.APIKey
export const apiKeySchema = z.object({
  id: z.string(),
  key_prefix: z.string(),
  key_hint: z.string(),
  name: z.string(),
  description: z.string().optional(),
  owner_id: z.string(),
  scopes: z.array(z.string()),
  expires_at: z.string().optional(),
  last_used_at: z.string().optional(),
  call_count: z.number().optional().default(0),
  last_source_ip: z.string().optional(),
  grace_period_expires_at: z.string().optional(),
  state: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});
export type APIKey = z.infer<typeof apiKeySchema>;

// GET /keys
export const apiKeyListSchema = pagedListSchema(apiKeySchema);
export const paginatedApiKeyListSchema = paginatedListSchema(apiKeySchema);
export type PaginatedAPIKeys = z.infer<typeof paginatedApiKeyListSchema>;

// core.Caveat
export const caveatSchema = z.object({
  predicate: z.string(),
});
export type Caveat = z.infer<typeof caveatSchema>;

// core.Macaroon
export const macaroonSchema = z.object({
  location: z.string(),
  key_id: z.string(),
  caveats: z.array(caveatSchema),
  signature: z.string(),
});
export type Macaroon = z.infer<typeof macaroonSchema>;

// POST /keys -> core.CreateKeyResponse
export const createKeyResponseSchema = z.object({
  api_key: apiKeySchema,
  raw_token: z.string(),
  macaroon: macaroonSchema,
});
export type CreateKeyResponse = z.infer<typeof createKeyResponseSchema>;

// POST /admin/keys/{id}/rotate -> core.RotateKeyResponse
export const rotateKeyResponseSchema = z.object({
  api_key: apiKeySchema,
  raw_token: z.string(),
  macaroon: macaroonSchema,
});
export type RotateKeyResponse = z.infer<typeof rotateKeyResponseSchema>;

// core.Scope
export const scopeSchema = z.object({
  name: z.string(),
  description: z.string().optional().default(""),
  created_at: z.string().optional(),
});
export const scopeListSchema = z.array(scopeSchema);
export type Scope = z.infer<typeof scopeSchema>;

// POST /keys/verify
export const verifyKeyResponseSchema = z.object({
  valid: z.boolean(),
  error: z.string().optional(),
});
export type VerifyKeyResponse = z.infer<typeof verifyKeyResponseSchema>;

// POST /keys/attenuate -> the attenuated core.Macaroon, returned bare.
export const attenuateResponseSchema = macaroonSchema;

