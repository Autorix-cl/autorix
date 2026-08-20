import { describe, expect, it } from "vitest";
import { ruleListSchema, ruleSchema, testMatchResponseSchema } from "./aegis";

describe("ruleSchema", () => {
  const validRule = {
    id: "rule-1",
    description: "Protect admin API",
    match: { url: "/admin/<.*>", methods: ["GET", "POST"] },
    authenticators: [{ handler: "bearer_token", config: {} }],
    authorizer: { handler: "allow", config: {} },
    mutators: [{ handler: "header", config: { name: "X-User" } }],
    upstream: { url: "http://admin-service:8080" },
  };

  it("accepts a realistic rule payload", () => {
    expect(ruleSchema.safeParse(validRule).success).toBe(true);
  });

  it("accepts a list response", () => {
    expect(ruleListSchema.safeParse([validRule]).success).toBe(true);
  });

  it("rejects a rule missing match.methods", () => {
    const payload = { ...validRule, match: { url: "/admin/<.*>" } };
    expect(ruleSchema.safeParse(payload).success).toBe(false);
  });
});

describe("testMatchResponseSchema", () => {
  it("accepts an unmatched response", () => {
    expect(testMatchResponseSchema.safeParse({ matched: false }).success).toBe(true);
  });

  it("rejects a response missing matched", () => {
    expect(testMatchResponseSchema.safeParse({ rule: null }).success).toBe(false);
  });
});
