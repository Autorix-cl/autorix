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

// core.Session
export const sessionSchema = z.object({
  id: z.string(),
  identity_id: z.string(),
  identity: identitySchema.optional(),
  token: z.string().optional(),
  expires_at: z.string(),
  authenticated_at: z.string(),
});
export type Session = z.infer<typeof sessionSchema>;

export const sessionListSchema = z.array(sessionSchema);
export const paginatedSessionListSchema = paginatedListSchema(sessionSchema);
export type PaginatedSessions = z.infer<typeof paginatedSessionListSchema>;

// Credential Inspection (administrative view without hashes)
export const credentialInspectionSchema = z.object({
  id: z.string(),
  identity_id: z.string(),
  credential_type: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
  force_rotation: z.boolean().optional(),
});
export const credentialListSchema = z.array(credentialInspectionSchema);
export type CredentialInspection = z.infer<typeof credentialInspectionSchema>;

// Administrative Password Reset Result
export const resetPasswordResultSchema = z.object({
  status: z.string(),
  temporary_password: z.string().optional().nullable(),
  force_rotation: z.boolean(),
});
export type ResetPasswordResult = z.infer<typeof resetPasswordResultSchema>;

// Administrative Recovery Link Result
export const recoveryLinkResultSchema = z.object({
  recovery_link: z.string(),
  token: z.string(),
  expires_at: z.string(),
});
export type RecoveryLinkResult = z.infer<typeof recoveryLinkResultSchema>;

// MFA Status
export const mfaStatusSchema = z.object({
  totp_enabled: z.boolean(),
  confirmed: z.boolean().optional(),
  backup_codes_remaining: z.number().optional(),
  webauthn_enabled: z.boolean().optional(),
});
export type MfaStatus = z.infer<typeof mfaStatusSchema>;

// Identity Trait JSON Schema Definition
export const identitySchemaDefinitionSchema = z.object({
  id: z.string(),
  name: z.string(),
  schema: z.record(z.string(), z.unknown()),
  version: z.number().optional(),
  created_at: z.string().optional(),
  updated_at: z.string().optional(),
});
export const identitySchemaListSchema = z.array(identitySchemaDefinitionSchema);
export type IdentitySchemaDefinition = z.infer<typeof identitySchemaDefinitionSchema>;

// Bulk Import Result
export const bulkImportResultSchema = z.object({
  total: z.number(),
  succeeded: z.number(),
  failed: z.number(),
  errors: z.array(
    z.object({
      row: z.number(),
      email: z.string().optional(),
      error: z.string(),
    })
  ),
});
export type BulkImportResult = z.infer<typeof bulkImportResultSchema>;

// POST /self-service/registration -> { session, identity }
export const registrationResponseSchema = z.object({
  session: sessionSchema,
  identity: identitySchema,
});

