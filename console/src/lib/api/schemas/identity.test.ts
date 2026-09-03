import { describe, expect, it } from "vitest";
import {
  identityListSchema,
  identitySchema,
  registrationResponseSchema,
  sessionListSchema,
  credentialListSchema,
  resetPasswordResultSchema,
  recoveryLinkResultSchema,
  mfaStatusSchema,
  identitySchemaListSchema,
  bulkImportResultSchema,
} from "./identity";

describe("identitySchema", () => {
  it("accepts a realistic identity payload", () => {
    const payload = {
      id: "b3f1b1a0-0000-4000-8000-000000000000",
      schema_id: "default",
      traits: { email: "alice@example.com", name: { first: "Alice", last: "Doe" } },
      state: "active",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    expect(identitySchema.safeParse(payload).success).toBe(true);
  });

  it("accepts a list response", () => {
    expect(identityListSchema.safeParse([]).success).toBe(true);
  });

  it("rejects an identity missing required fields", () => {
    const payload = { id: "b3f1b1a0-0000-4000-8000-000000000000", state: "active" };
    expect(identitySchema.safeParse(payload).success).toBe(false);
  });
});

describe("registrationResponseSchema", () => {
  it("accepts a realistic registration response", () => {
    const payload = {
      session: {
        id: "s-1",
        identity_id: "b3f1b1a0-0000-4000-8000-000000000000",
        token: "sess_tok_abc",
        expires_at: "2026-01-02T00:00:00Z",
        authenticated_at: "2026-01-01T00:00:00Z",
      },
      identity: {
        id: "b3f1b1a0-0000-4000-8000-000000000000",
        schema_id: "default",
        traits: { email: "alice@example.com" },
        state: "active",
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
    };
    expect(registrationResponseSchema.safeParse(payload).success).toBe(true);
  });

  it("rejects a response missing the identity", () => {
    const payload = {
      session: {
        id: "s-1",
        identity_id: "b3f1b1a0-0000-4000-8000-000000000000",
        expires_at: "2026-01-02T00:00:00Z",
        authenticated_at: "2026-01-01T00:00:00Z",
      },
    };
    expect(registrationResponseSchema.safeParse(payload).success).toBe(false);
  });
});

describe("session & credential schemas", () => {
  it("validates session lists", () => {
    const payload = [
      {
        id: "s-1",
        identity_id: "b3f1b1a0-0000-4000-8000-000000000000",
        expires_at: "2026-01-02T00:00:00Z",
        authenticated_at: "2026-01-01T00:00:00Z",
      },
    ];
    expect(sessionListSchema.safeParse(payload).success).toBe(true);
  });

  it("validates credential inspection list without exposing hashes", () => {
    const payload = [
      {
        id: "c-1",
        identity_id: "b3f1b1a0-0000-4000-8000-000000000000",
        credential_type: "password",
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
        force_rotation: true,
      },
    ];
    expect(credentialListSchema.safeParse(payload).success).toBe(true);
  });

  it("validates reset password and recovery link results", () => {
    const resetResult = {
      status: "password_reset",
      temporary_password: "temp-pass-1234",
      force_rotation: true,
    };
    expect(resetPasswordResultSchema.safeParse(resetResult).success).toBe(true);

    const recoveryResult = {
      recovery_link: "/self-service/recovery?token=tok-123",
      token: "tok-123",
      expires_at: "2026-01-01T01:00:00Z",
    };
    expect(recoveryLinkResultSchema.safeParse(recoveryResult).success).toBe(true);
  });

  it("validates MFA status and schema definitions", () => {
    const mfa = {
      totp_enabled: true,
      confirmed: true,
      backup_codes_remaining: 8,
    };
    expect(mfaStatusSchema.safeParse(mfa).success).toBe(true);

    const schemas = [
      {
        id: "default",
        name: "Default User Schema",
        schema: { type: "object", properties: { email: { type: "string" } } },
        version: 1,
      },
    ];
    expect(identitySchemaListSchema.safeParse(schemas).success).toBe(true);

    const bulkResult = {
      total: 10,
      succeeded: 8,
      failed: 2,
      errors: [{ row: 4, email: "invalid", error: "Invalid email" }],
    };
    expect(bulkImportResultSchema.safeParse(bulkResult).success).toBe(true);
  });
});


