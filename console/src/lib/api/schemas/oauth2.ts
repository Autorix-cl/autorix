/**
 * Zod schemas for Janus's OAuth2 admin REST API
 * (janus/internal/transport/http/server.go, structs in janus/internal/core
 * and janus/internal/jwks). Field names mirror the Go json tags exactly
 * (snake_case).
 */
import { z } from "zod";

// core.OAuth2Client
export const oauth2ClientSchema = z.object({
  client_id: z.string(),
  client_name: z.string(),
  grant_types: z.array(z.string()),
  response_types: z.array(z.string()),
  redirect_uris: z.array(z.string()),
  scopes: z.array(z.string()),
  is_public: z.boolean(),
  created_at: z.string(),
  updated_at: z.string(),
});
export type OAuth2Client = z.infer<typeof oauth2ClientSchema>;

// GET /admin/clients
export const oauth2ClientListSchema = z.array(oauth2ClientSchema);

// jwks.JWK (RFC 7517)
export const jwkSchema = z.object({
  kty: z.string(),
  use: z.string(),
  alg: z.string(),
  kid: z.string(),
  n: z.string(),
  e: z.string(),
});

// GET /.well-known/jwks.json -> jwks.JWKS
export const jwksSchema = z.object({
  keys: z.array(jwkSchema),
});
