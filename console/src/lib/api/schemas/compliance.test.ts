import { describe, it, expect } from "vitest";
import {
  complianceEvidenceItemSchema,
  compliancePackageRequestSchema,
  compliancePackageResponseSchema,
} from "./compliance";

describe("Compliance Schemas (P8-S4)", () => {
  it("validates compliance evidence item schema", () => {
    const item = {
      id: "ctrl_soc2_cc6_1",
      framework: "SOC 2 Type II",
      control_id: "CC6.1",
      control_name: "Logical Access Controls & Authentication",
      status: "compliant" as const,
      evidence_type: "mfa_and_credential_vaulting",
      description: "Argon2id and MFA enforced",
      engine: "Ego",
      last_evaluated_at: "2026-08-18T20:00:00Z",
      evaluator: "argus-continuous-compliance",
      artifacts_count: 3,
      details: { memory_kb: 65536 },
    };
    expect(complianceEvidenceItemSchema.safeParse(item).success).toBe(true);
  });

  it("validates compliance package request schema", () => {
    const req = {
      framework: "SOC 2 Type II",
      period: "Q3-2026",
      include_audit_trail: true,
      include_access_review: true,
      include_cryptographic_proof: true,
    };
    expect(compliancePackageRequestSchema.safeParse(req).success).toBe(true);
  });

  it("validates compliance package response schema", () => {
    const res = {
      package_id: "pkg_123",
      framework: "SOC 2 Type II",
      period_start: "2026-07-01T00:00:00Z",
      period_end: "2026-09-30T23:59:59Z",
      status: "generated",
      generated_at: "2026-08-18T20:00:00Z",
      summary: {
        total_controls: 24,
        passing_controls: 24,
        failing_controls: 0,
        review_required: 0,
      },
    };
    expect(compliancePackageResponseSchema.safeParse(res).success).toBe(true);
  });
});
