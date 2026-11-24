import { describe, expect, it } from "vitest";

// eslint-disable-next-line @typescript-eslint/no-require-imports
const nextConfig = require("../next.config");

describe("security headers", () => {
  it("applies baseline browser protections to every route", async () => {
    const rules = await nextConfig.headers();
    expect(rules).toHaveLength(1);
    expect(rules[0].source).toBe("/(.*)");

    const headers = Object.fromEntries(
      rules[0].headers.map(({ key, value }: { key: string; value: string }) => [key, value]),
    );
    expect(headers["Content-Security-Policy"]).toBe(
      "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'; object-src 'none'",
    );
    expect(headers["Cross-Origin-Opener-Policy"]).toBe("same-origin-allow-popups");
    expect(headers["Cross-Origin-Resource-Policy"]).toBe("same-origin");
    expect(headers["Referrer-Policy"]).toBe("strict-origin-when-cross-origin");
    expect(headers["X-Content-Type-Options"]).toBe("nosniff");
    expect(headers["X-Frame-Options"]).toBe("DENY");
    expect(headers["Permissions-Policy"]).toBe("camera=(), geolocation=(), microphone=()");
    expect(headers["Strict-Transport-Security"]).toContain("includeSubDomains");
  });
});
