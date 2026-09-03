/**
 * Zod schemas for Janus's OAuth2 admin REST API
 * (janus/internal/transport/http/server.go, structs in janus/internal/core
 * and janus/internal/jwks). Field names mirror the Go json tags exactly
 * (snake_case).
 */
import { z } from "zod";
import { pagedListSchema } from "../schema";

// core.OAuth2Client
export const oauth2ClientSchema = z.object({
  client_id: z.string(),
  client_name: z.string(),
  grant_types: z.array(z.string()),
  response_types: z.array(z.string()),
  redirect_uris: z.array(z.string()),
  scopes: z.array(z.string()),
  is_public: z.boolean(),
  previous_secret_expires_at: z.string().nullable().optional(),
  has_previous_secret: z.boolean().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});
export type OAuth2Client = z.infer<typeof oauth2ClientSchema>;

// GET /admin/clients
export const oauth2ClientListSchema = pagedListSchema(oauth2ClientSchema);

// POST /admin/clients/{id}/rotate-secret
export const rotateSecretResponseSchema = z.object({
  client_id: z.string(),
  client_secret: z.string(),
  previous_secret_expires_at: z.string().nullable().optional(),
});
export type RotateSecretResponse = z.infer<typeof rotateSecretResponseSchema>;

// Scope Catalogue
export const oauth2ScopeSchema = z.object({
  name: z.string(),
  description: z.string().default(""),
  claims: z.array(z.string()).default([]),
  created_at: z.string().optional(),
});
export type OAuth2Scope = z.infer<typeof oauth2ScopeSchema>;

export const oauth2ScopeListSchema = pagedListSchema(oauth2ScopeSchema);

export const createScopeRequestSchema = z.object({
  name: z.string().min(1),
  description: z.string().default(""),
  claims: z.array(z.string()).default([]),
});
export type CreateScopeRequest = z.infer<typeof createScopeRequestSchema>;

// RFC 7662 Token Introspection
export const introspectRequestSchema = z.object({
  token: z.string().min(1),
  token_type_hint: z.string().optional(),
});
export type IntrospectRequest = z.infer<typeof introspectRequestSchema>;

export const introspectResponseSchema = z.object({
  active: z.boolean(),
  scope: z.string().optional(),
  client_id: z.string().optional(),
  sub: z.string().optional(),
  exp: z.number().optional(),
  iat: z.number().optional(),
  iss: z.string().optional(),
  token_type: z.string().optional(),
});
export type IntrospectResponse = z.infer<typeof introspectResponseSchema>;

// RFC 7009 Token Revocation
export const revokeTokenRequestSchema = z.object({
  token: z.string().min(1),
  token_type_hint: z.string().optional(),
});
export type RevokeTokenRequest = z.infer<typeof revokeTokenRequestSchema>;

export const revokeTokenResponseSchema = z.object({
  status: z.string(),
});
export type RevokeTokenResponse = z.infer<typeof revokeTokenResponseSchema>;

// Key Rollover
export const rotateKeysResponseSchema = z.object({
  status: z.string(),
  new_kid: z.string().optional(),
  active_keys_count: z.number().optional(),
});
export type RotateKeysResponse = z.infer<typeof rotateKeysResponseSchema>;

// Grants & Consents
export const oauth2GrantSchema = z.object({
  grant_id: z.string(),
  client_id: z.string(),
  subject: z.string(),
  scopes: z.array(z.string()),
  created_at: z.string(),
  expires_at: z.string().nullable().optional(),
});
export type OAuth2Grant = z.infer<typeof oauth2GrantSchema>;

export const oauth2GrantListSchema = pagedListSchema(oauth2GrantSchema);

// jwks.JWK (RFC 7517)
export const jwkSchema = z.object({
  kty: z.string(),
  use: z.string(),
  alg: z.string(),
  kid: z.string(),
  n: z.string(),
  e: z.string(),
});
export type JWK = z.infer<typeof jwkSchema>;

// GET /.well-known/jwks.json -> jwks.JWKS
export const jwksSchema = z.object({
  keys: z.array(jwkSchema),
});
export type JWKS = z.infer<typeof jwksSchema>;

