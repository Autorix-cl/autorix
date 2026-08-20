import { describe, it, expect } from "vitest";
import { healthResponseSchema } from "./health";

describe("healthResponseSchema", () => {
  it("accepts a real aggregator payload", () => {
    const result = healthResponseSchema.safeParse({
      checkedAt: "2026-08-18T10:00:00.000Z",
      services: [
        { key: "ego", name: "Autorix Ego", port: 4433, protocol: "REST", status: "healthy", latencyMs: 12.4 },
        {
          key: "nexus",
          name: "Autorix Nexus",
          port: 8080,
          protocol: "gRPC (REST admin)",
          status: "unreachable",
          latencyMs: null,
        },
      ],
    });
    expect(result.success).toBe(true);
  });

  it("rejects an unknown status value", () => {
    const result = healthResponseSchema.safeParse({
      checkedAt: "2026-08-18T10:00:00.000Z",
      services: [
        { key: "ego", name: "Autorix Ego", port: 4433, protocol: "REST", status: "totally-fine", latencyMs: 1 },
      ],
    });
    expect(result.success).toBe(false);
  });

  it("requires latencyMs to be present (number or null), not omitted", () => {
    const result = healthResponseSchema.safeParse({
      checkedAt: "2026-08-18T10:00:00.000Z",
      services: [{ key: "ego", name: "Autorix Ego", port: 4433, protocol: "REST", status: "healthy" }],
    });
    expect(result.success).toBe(false);
  });
});
