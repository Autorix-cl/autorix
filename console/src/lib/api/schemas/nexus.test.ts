import { describe, expect, it } from "vitest";
import { checkResponseSchema, tupleListSchema, tupleSchema } from "./nexus";

describe("tupleSchema", () => {
  it("accepts a realistic tuple payload", () => {
    const payload = {
      namespace: "document",
      object: "doc:42",
      relation: "viewer",
      subject_namespace: "user",
      subject_id: "user:alice",
      subject_relation: "",
      caveat_name: "expires_at",
      caveat_context: { expires_at: "2026-01-01T00:00:00Z" },
    };
    expect(tupleSchema.safeParse(payload).success).toBe(true);
  });

  it("accepts a list response", () => {
    expect(
      tupleListSchema.safeParse([
        {
          namespace: "document",
          object: "doc:42",
          relation: "viewer",
          subject_namespace: "user",
          subject_id: "user:alice",
        },
      ]).success,
    ).toBe(true);
  });

  it("rejects a tuple missing required fields", () => {
    const payload = { namespace: "document", object: "doc:42" };
    expect(tupleSchema.safeParse(payload).success).toBe(false);
  });
});

describe("checkResponseSchema", () => {
  it("accepts a valid check response", () => {
    expect(checkResponseSchema.safeParse({ allowed: true, reason: "matched" }).success).toBe(true);
  });

  it("rejects a response missing allowed", () => {
    expect(checkResponseSchema.safeParse({ reason: "matched" }).success).toBe(false);
  });
});
