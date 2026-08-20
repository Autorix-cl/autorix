import { describe, it, expect } from "vitest";
import { redactSecrets, redactObject } from "./redact";

describe("Secret Redaction Discipline (P3-S5-T6)", () => {
  it("redacts Argus and Vulcan tokens from text", () => {
    const text = "Connecting with token ast_0123456789abcdef0123456789abcdef and key av_live_abcdef1234567890abcdef1234";
    const redacted = redactSecrets(text);
    expect(redacted).not.toContain("ast_0123456789abcdef0123456789abcdef");
    expect(redacted).not.toContain("av_live_abcdef1234567890abcdef1234");
    expect(redacted).toContain("[REDACTED_SECRET]");
  });

  it("redacts Argon2id hashes and Bearer tokens", () => {
    const text = "Hash: $argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHQ$aGFzaGhhc2g and header Bearer eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0";
    const redacted = redactSecrets(text);
    expect(redacted).not.toContain("$argon2id$");
    expect(redacted).toContain("[REDACTED_SECRET]");
  });

  it("recursively redacts sensitive keys from objects", () => {
    const payload = {
      user: "admin",
      password: "SuperSecretPassword123!",
      session_token: "ast_0123456789abcdef0123456789abcdef",
      nested: {
        api_secret: "secret-key",
        normalField: "public-value",
      },
    };

    const redacted = redactObject(payload);
    expect(redacted.password).toBe("[REDACTED]");
    expect(redacted.session_token).toBe("[REDACTED]");
    expect(redacted.nested.api_secret).toBe("[REDACTED]");
    expect(redacted.nested.normalField).toBe("public-value");
  });
});
