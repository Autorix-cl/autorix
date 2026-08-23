import { z } from "zod";

export const CreateKeySchema = z.object({
  name: z.string().min(3, "Name must be at least 3 characters").max(64, "Name is too long"),
  expires_in: z.enum(["7d", "30d", "90d", "never"]),
  scopes: z.array(z.string()).min(1, "Select at least one scope"),
});

export type CreateKeyInput = z.infer<typeof CreateKeySchema>;

export const KeyMetadataSchema = z.object({
  id: z.string(),
  name: z.string(),
  prefix: z.string(),
  scopes: z.array(z.string()),
  created_at: z.string(),
  expires_at: z.string().nullable(),
  revoked: z.boolean(),
});

export type KeyMetadata = z.infer<typeof KeyMetadataSchema>;

// Internal type for DB which includes the secret
export type RootKeyRecord = KeyMetadata & {
  secret: string; // Stored securely in DB, required for Macaroon HMAC verification
};
