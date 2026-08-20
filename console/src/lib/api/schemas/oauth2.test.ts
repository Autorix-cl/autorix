import { describe, expect, it } from "vitest";
import { jwksSchema, oauth2ClientListSchema, oauth2ClientSchema } from "./oauth2";

describe("oauth2ClientSchema", () => {
  it("accepts a realistic client payload", () => {
    const payload = {
      client_id: "console-app",
      client_name: "Console App",
      grant_types: ["authorization_code", "refresh_token"],
      response_types: ["code"],
      redirect_uris: ["https://console.example.com/callback"],
      scopes: ["openid", "profile"],
      is_public: true,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    expect(oauth2ClientSchema.safeParse(payload).success).toBe(true);
  });

  it("accepts a list response", () => {
    expect(oauth2ClientListSchema.safeParse([]).success).toBe(true);
  });

  it("rejects a client missing required fields", () => {
    const payload = { client_id: "console-app" };
    expect(oauth2ClientSchema.safeParse(payload).success).toBe(false);
  });
});

describe("jwksSchema", () => {
  it("accepts a realistic jwks payload", () => {
    const payload = {
      keys: [{ kty: "RSA", use: "sig", alg: "RS256", kid: "key-1", n: "abc123", e: "AQAB" }],
    };
    expect(jwksSchema.safeParse(payload).success).toBe(true);
  });

  it("rejects a jwks payload with a malformed key", () => {
    const payload = { keys: [{ kty: "RSA", use: "sig" }] };
    expect(jwksSchema.safeParse(payload).success).toBe(false);
  });
});
