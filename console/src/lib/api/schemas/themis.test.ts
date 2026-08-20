import { describe, expect, it } from "vitest";
import { evaluateResponseSchema, policyListSchema, policySchema } from "./themis";

describe("policySchema", () => {
  it("accepts a realistic PascalCase policy payload", () => {
    const payload = {
      ID: "pol_123",
      TenantID: "default",
      Name: "require-mfa",
      Description: "Require MFA for admin actions",
      Expression: "request.mfa == true",
      Priority: 1,
      Enabled: true,
      Labels: { team: "platform" },
      CreatedAt: "2026-01-01T00:00:00Z",
      UpdatedAt: "2026-01-01T00:00:00Z",
    };
    expect(policySchema.safeParse(payload).success).toBe(true);
  });

  it("accepts a list response", () => {
    expect(
      policyListSchema.safeParse([
        {
          ID: "pol_123",
          TenantID: "default",
          Name: "require-mfa",
          Expression: "request.mfa == true",
          Priority: 1,
          Enabled: true,
          CreatedAt: "2026-01-01T00:00:00Z",
          UpdatedAt: "2026-01-01T00:00:00Z",
        },
      ]).success,
    ).toBe(true);
  });

  it("rejects a snake_case payload (would silently pass as {} otherwise)", () => {
    const payload = {
      id: "pol_123",
      tenant_id: "default",
      name: "require-mfa",
      expression: "request.mfa == true",
      priority: 1,
      enabled: true,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    expect(policySchema.safeParse(payload).success).toBe(false);
  });
});

describe("evaluateResponseSchema", () => {
  it("accepts a realistic evaluate response", () => {
    const payload = {
      AllPassed: false,
      Results: [
        { PolicyID: "pol_123", PolicyName: "require-mfa", Passed: false, Error: "", Expression: "request.mfa == true" },
      ],
      TotalEvaluated: 1,
    };
    expect(evaluateResponseSchema.safeParse(payload).success).toBe(true);
  });

  it("rejects a response missing AllPassed", () => {
    expect(evaluateResponseSchema.safeParse({ Results: [], TotalEvaluated: 0 }).success).toBe(false);
  });
});
