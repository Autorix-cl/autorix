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

describe("Themis Engine Depth Schemas", () => {
  it("validates GoPolicy schema", async () => {
    const { goPolicySchema, goPolicyListSchema } = await import("./themis");
    const payload = {
      id: "pol_1",
      tenant_id: "default",
      name: "require-mfa",
      expression: "request.mfa == true",
      priority: 10,
      enabled: true,
      created_at: "2026-09-02T10:00:00Z",
    };
    expect(goPolicySchema.safeParse(payload).success).toBe(true);
    expect(goPolicyListSchema.safeParse([payload]).success).toBe(true);
  });

  it("validates PolicyVersion schema", async () => {
    const { policyVersionSchema, policyVersionListSchema } = await import("./themis");
    const payload = {
      id: "pv_1",
      policy_id: "pol_1",
      tenant_id: "default",
      version: 2,
      name: "require-mfa-v2",
      expression: "request.mfa == true && request.auth.role == 'admin'",
      priority: 5,
      enabled: true,
      created_at: "2026-09-02T11:00:00Z",
    };
    expect(policyVersionSchema.safeParse(payload).success).toBe(true);
    expect(policyVersionListSchema.safeParse([payload]).success).toBe(true);
  });

  it("validates PolicyFixture and TestSuiteResult schemas", async () => {
    const { policyFixtureSchema, testSuiteResultSchema } = await import("./themis");
    const fixture = {
      id: "fix_1",
      policy_id: "pol_1",
      tenant_id: "default",
      name: "admin with mfa allowed",
      payload: { request: { mfa: true } },
      expected_result: true,
      created_at: "2026-09-02T10:00:00Z",
      updated_at: "2026-09-02T10:00:00Z",
    };
    expect(policyFixtureSchema.safeParse(fixture).success).toBe(true);

    const testSuite = {
      policy_id: "pol_1",
      all_passed: true,
      total_tests: 2,
      passed_tests: 2,
      failed_tests: 0,
      results: [
        {
          fixture_id: "fix_1",
          fixture_name: "admin with mfa allowed",
          expected_result: true,
          actual_result: true,
          passed: true,
          error: "",
        },
      ],
    };
    expect(testSuiteResultSchema.safeParse(testSuite).success).toBe(true);
  });

  it("validates ValidationResult and DryRunResult schemas", async () => {
    const { validationResultSchema, dryRunResultSchema } = await import("./themis");
    const validation = {
      valid: true,
      variables: ["request.mfa", "request.auth.role"],
      errors: [],
    };
    expect(validationResultSchema.safeParse(validation).success).toBe(true);

    const dryRun = {
      passed: true,
      error: "",
    };
    expect(dryRunResultSchema.safeParse(dryRun).success).toBe(true);
  });
});

