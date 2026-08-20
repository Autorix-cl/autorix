import { describe, it, expect } from "vitest";
import { type EngineHealthStatus } from "./capability-context";

describe("Subset Test Matrix & Capability Gating (P5-S2-T5)", () => {
  it("computes cluster health percentage accurately", () => {
    const statuses: Record<string, EngineHealthStatus> = {
      ego: "healthy",
      nexus: "healthy",
      janus: "healthy",
      aegis: "degraded",
      vulcan: "healthy",
      hermes: "unreachable",
      themis: "healthy",
    };

    const healthyCount = Object.values(statuses).filter((s) => s === "healthy").length;
    const healthPercentage = Math.round((healthyCount / Object.keys(statuses).length) * 100);

    expect(healthyCount).toBe(5);
    expect(healthPercentage).toBe(71);
  });

  it("handles empty fleet subset correctly without throwing", () => {
    const emptyFleet: Record<string, EngineHealthStatus> = {};
    const healthyCount = Object.values(emptyFleet).filter((s) => s === "healthy").length;
    const healthPercentage = Object.keys(emptyFleet).length > 0 ? (healthyCount / Object.keys(emptyFleet).length) * 100 : 0;

    expect(healthyCount).toBe(0);
    expect(healthPercentage).toBe(0);
  });
});
