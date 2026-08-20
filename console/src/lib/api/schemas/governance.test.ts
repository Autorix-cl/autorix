import { describe, it, expect } from "vitest";
import {
  organisationSchema,
  projectSchema,
  environmentSchema,
} from "./governance";

describe("Governance Schemas (P8-S3)", () => {
  it("validates organisation schema", () => {
    const org = {
      id: "org_123",
      name: "Acme Corp",
      slug: "acme-corp",
      created_at: "2026-08-18T20:00:00Z",
    };
    expect(organisationSchema.safeParse(org).success).toBe(true);
  });

  it("validates project schema", () => {
    const project = {
      id: "prj_456",
      org_id: "org_123",
      name: "Core Services",
      slug: "core-services",
    };
    expect(projectSchema.safeParse(project).success).toBe(true);
  });

  it("validates environment schema", () => {
    const env = {
      id: "env_789",
      project_id: "prj_456",
      name: "Production",
      slug: "prod",
      tier: "production" as const,
      is_production: true,
    };
    expect(environmentSchema.safeParse(env).success).toBe(true);
  });
});
