import { describe, expect, it } from "vitest";

// eslint-disable-next-line @typescript-eslint/no-require-imports
const nextConfig = require("../next.config");

describe("security headers", () => {
  it("applies baseline browser protections to every route", async () => {
    const rules = await nextConfig.headers();
    expect(rules).toHaveLength(1);
    expect(rules[0].source).toBe("/(.*)");

    const headers = Object.fromEntries(rules[0].headers.map(({ key, value }: { key: string; value: string }) => [key, value]));
    expect(headers["Content-Security-Policy"]).toContain("frame-ancestors 'none'");
    expect(headers["X-Content-Type-Options"]).toBe("nosniff");
    expect(headers["X-Frame-Options"]).toBe("DENY");
    expect(headers["Strict-Transport-Security"]).toContain("includeSubDomains");
  });
});
