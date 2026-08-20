/**
 * Zod schemas for Hermes's SAML/SCIM REST API
 * (hermes/internal/transport/http/server.go, structs in hermes/internal/core).
 * Field names mirror the Go json tags exactly. SCIM resources use SCIM's own
 * camelCase conventions (RFC 7643/7644), not the console's usual snake_case.
 */
import { z } from "zod";
import { pagedListSchema } from "../schema";

// core.SAMLProvider
export const samlProviderSchema = z.object({
  id: z.string(),
  display_name: z.string(),
  idp_entity_id: z.string(),
  idp_sso_url: z.string(),
  idp_certificate_pem: z.string(),
  sp_entity_id: z.string(),
  attribute_mapping: z.record(z.string(), z.string()),
  enabled: z.boolean(),
  created_at: z.string(),
  updated_at: z.string(),
});
export type SAMLProvider = z.infer<typeof samlProviderSchema>;

// GET /admin/saml/providers
export const samlProviderListSchema = pagedListSchema(samlProviderSchema);

// core.SCIMEmail
export const scimEmailSchema = z.object({
  value: z.string(),
  type: z.string(),
  primary: z.boolean(),
});

// core.SCIMMeta
export const scimMetaSchema = z.object({
  resourceType: z.string(),
  created: z.string(),
  lastModified: z.string(),
  location: z.string(),
});

// core.SCIMUser, as returned by scimEngine.FormatUser
export const scimUserSchema = z.object({
  schemas: z.array(z.string()),
  id: z.string(),
  externalId: z.string().optional(),
  userName: z.string(),
  displayName: z.string().optional(),
  emails: z.array(scimEmailSchema),
  active: z.boolean(),
  meta: scimMetaSchema,
});
export type SCIMUser = z.infer<typeof scimUserSchema>;

// GET /scim/v2/Users -> core.SCIMListResponse (RFC 7644 paginated list, NOT a
// bare array: Resources is wrapped alongside pagination metadata).
export const scimListResponseSchema = z.object({
  schemas: z.array(z.string()),
  totalResults: z.number(),
  startIndex: z.number(),
  itemsPerPage: z.number(),
  Resources: z.array(scimUserSchema),
});
