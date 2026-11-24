import { afterEach, describe, expect, it } from "vitest";
import { buildInternalAuthHeaders, getInternalAuthToken } from "./internal-headers";

describe("internal-headers", () => {
  const originalInternal = process.env.AUTORIX_INTERNAL_AUTH_SECRET;
  const originalConsole = process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN;

  afterEach(() => {
    if (originalInternal === undefined) {
      delete process.env.AUTORIX_INTERNAL_AUTH_SECRET;
    } else {
      process.env.AUTORIX_INTERNAL_AUTH_SECRET = originalInternal;
    }
    if (originalConsole === undefined) {
      delete process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN;
    } else {
      process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN = originalConsole;
    }
  });

  it("omits token headers when neither env var is configured", () => {
    delete process.env.AUTORIX_INTERNAL_AUTH_SECRET;
    delete process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN;

    expect(getInternalAuthToken()).toBeUndefined();
    expect(buildInternalAuthHeaders({ "Content-Type": "application/json" })).toEqual({
      "x-autorix-caller": "console",
      "Content-Type": "application/json",
    });
  });

  it("prefers AUTORIX_INTERNAL_AUTH_SECRET and sets both internal headers", () => {
    process.env.AUTORIX_INTERNAL_AUTH_SECRET = "cluster-secret";
    process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN = "legacy-secret";

    expect(getInternalAuthToken()).toBe("cluster-secret");
    expect(buildInternalAuthHeaders()).toEqual({
      "x-autorix-caller": "console",
      "x-autorix-internal-token": "cluster-secret",
      "x-autorix-console-token": "cluster-secret",
    });
  });

  it("falls back to AUTORIX_CONSOLE_INTERNAL_TOKEN", () => {
    delete process.env.AUTORIX_INTERNAL_AUTH_SECRET;
    process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN = "legacy-secret";

    expect(getInternalAuthToken()).toBe("legacy-secret");
    expect(buildInternalAuthHeaders()).toEqual({
      "x-autorix-caller": "console",
      "x-autorix-internal-token": "legacy-secret",
      "x-autorix-console-token": "legacy-secret",
    });
  });
});
