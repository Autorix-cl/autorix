import { describe, it, expect } from "vitest";
import {
  auditEntrySchema,
  auditVerificationSchema,
  auditExportResponseSchema,
} from "./audit";

describe("Audit Schemas (P8-S1)", () => {
  it("validates a complete audit log entry", () => {
    const entry = {
      id: "aud_123",
      timestamp: "2026-08-18T20:00:00Z",
      actor: "admin@enterprise.io",
      actor_type: "operator",
      action: "UPDATE",
      resource_type: "policy",
      resource_id: "pol_themis_1",
      environment: "production",
      request_id: "req_xyz",
      ip_address: "127.0.0.1",
      user_agent: "Console/1.0",
      outcome: "success",
      before_state: { priority: 5 },
      after_state: { priority: 1 },
      hash: "a4f8e91c7b3d2e0f8a9c1b2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f",
      prev_hash: "9b3c4d5e6f7a8b9c0d1e2f3a4f8e91c7b3d2e0f8a9c1b2d3e4f5a6b7c8d9e0f1",
      sequence: 100,
    };

    const parsed = auditEntrySchema.safeParse(entry);
    expect(parsed.success).toBe(true);
  });

  it("validates audit verification result", () => {
    const verification = {
      verified: true,
      chain_length: 1042,
      head_hash: "a4f8e91c7b3d2e0f8a9c1b2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f",
      genesis_hash: "0000000000000000000000000000000000000000000000000000000000000000",
      verified_at: "2026-08-18T20:00:00Z",
      algorithm: "SHA-256",
    };

    const parsed = auditVerificationSchema.safeParse(verification);
    expect(parsed.success).toBe(true);
  });

  it("validates audit export response", () => {
    const exportData = {
      format: "csv",
      content: "id,timestamp,actor\naud_123,2026-08-18T20:00:00Z,admin",
      filename: "audit-export.csv",
      record_count: 1,
    };

    const parsed = auditExportResponseSchema.safeParse(exportData);
    expect(parsed.success).toBe(true);
  });
});
