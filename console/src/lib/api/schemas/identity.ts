import { z } from "zod";
import { pagedListSchema, paginatedListSchema } from "../schema";

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

// GET /admin/identities (legacy flat array)
export const identityListSchema = pagedListSchema(identitySchema);

// GET /admin/identities (paginated envelope)
export const paginatedIdentityListSchema = paginatedListSchema(identitySchema);
export type PaginatedIdentities = z.infer<typeof paginatedIdentityListSchema>;

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
