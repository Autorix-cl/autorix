import { describe, expect, it } from "vitest";
import { identityListSchema, identitySchema, registrationResponseSchema } from "./identity";

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
