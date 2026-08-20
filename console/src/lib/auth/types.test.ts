import { describe, it, expect } from "vitest";
import { hasPermission } from "./types";

describe("Role Permissions (P3-S4)", () => {
  it("owner has wildcard permission for everything", () => {
    expect(hasPermission("owner", "identities:write")).toBe(true);
    expect(hasPermission("owner", "fleet:manage")).toBe(true);
    expect(hasPermission("owner", "custom:action")).toBe(true);
    expect(hasPermission("owner", "audit:write")).toBe(true);
    expect(hasPermission("owner", "governance:write")).toBe(true);
    expect(hasPermission("owner", "compliance:write")).toBe(true);
  });

  it("admin has full management privileges", () => {
    expect(hasPermission("admin", "identities:write")).toBe(true);
    expect(hasPermission("admin", "oauth2:write")).toBe(true);
    expect(hasPermission("admin", "fleet:manage")).toBe(true);
    expect(hasPermission("admin", "audit:read")).toBe(true);
    expect(hasPermission("admin", "audit:write")).toBe(true);
    expect(hasPermission("admin", "governance:write")).toBe(true);
    expect(hasPermission("admin", "compliance:write")).toBe(true);
  });

  it("operator has read/write on resources but not fleet admin", () => {
    expect(hasPermission("operator", "identities:read")).toBe(true);
    expect(hasPermission("operator", "identities:write")).toBe(true);
    expect(hasPermission("operator", "fleet:read")).toBe(true);
    expect(hasPermission("operator", "fleet:manage")).toBe(false);
    expect(hasPermission("operator", "audit:read")).toBe(true);
    expect(hasPermission("operator", "governance:read")).toBe(true);
    expect(hasPermission("operator", "compliance:read")).toBe(true);
  });

  it("auditor is strictly read-only", () => {
    expect(hasPermission("auditor", "identities:read")).toBe(true);
    expect(hasPermission("auditor", "audit:read")).toBe(true);
    expect(hasPermission("auditor", "governance:read")).toBe(true);
    expect(hasPermission("auditor", "compliance:read")).toBe(true);
    expect(hasPermission("auditor", "identities:write")).toBe(false);
    expect(hasPermission("auditor", "fleet:manage")).toBe(false);
    expect(hasPermission("auditor", "audit:write")).toBe(false);
  });
});
