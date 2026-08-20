/**
 * Zod schemas for Vulcan's API Key / Macaroon REST API
 * (vulcan/internal/transport/http/server.go, structs in vulcan/internal/core).
 * Field names mirror the Go json tags exactly (snake_case). Pointer fields
 * with `omitempty` (expires_at, last_used_at) are omitted entirely by Go's
 * encoder when nil, so they're modeled as `.optional()`, not `.nullable()`.
 */
import { z } from "zod";
import { pagedListSchema } from "../schema";

// core.APIKey
export const apiKeySchema = z.object({
  id: z.string(),
  key_prefix: z.string(),
  key_hint: z.string(),
  name: z.string(),
  owner_id: z.string(),
  scopes: z.array(z.string()),
  expires_at: z.string().optional(),
  last_used_at: z.string().optional(),
  state: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});
export type APIKey = z.infer<typeof apiKeySchema>;

// GET /keys
export const apiKeyListSchema = pagedListSchema(apiKeySchema);

// core.Caveat
export const caveatSchema = z.object({
  predicate: z.string(),
});

// core.Macaroon
export const macaroonSchema = z.object({
  location: z.string(),
  key_id: z.string(),
  caveats: z.array(caveatSchema),
  signature: z.string(),
});

// POST /keys -> core.CreateKeyResponse
export const createKeyResponseSchema = z.object({
  api_key: apiKeySchema,
  raw_token: z.string(),
  macaroon: macaroonSchema,
});

// POST /keys/attenuate -> the attenuated core.Macaroon, returned bare.
export const attenuateResponseSchema = macaroonSchema;
