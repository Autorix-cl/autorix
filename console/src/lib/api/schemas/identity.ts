/**
 * Zod schemas for Ego's Identity REST API (ego/internal/transport/http/server.go,
 * structs defined in ego/internal/core). Field names mirror the Go json tags
 * exactly (snake_case).
 */
import { z } from "zod";

// core.Identity
export const identitySchema = z.object({
  id: z.string(),
  schema_id: z.string(),
  traits: z.record(z.string(), z.unknown()),
  state: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});
export type Identity = z.infer<typeof identitySchema>;

// GET /admin/identities
export const identityListSchema = z.array(identitySchema);

// core.Session, as embedded in the registration response. Token is only
// present at creation time.
export const sessionSchema = z.object({
  id: z.string(),
  identity_id: z.string(),
  identity: identitySchema.optional(),
  token: z.string().optional(),
  expires_at: z.string(),
  authenticated_at: z.string(),
});

// POST /self-service/registration -> { session, identity }
export const registrationResponseSchema = z.object({
  session: sessionSchema,
  identity: identitySchema,
});
