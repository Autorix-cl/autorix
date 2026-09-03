import { describe, expect, it } from "vitest";
import {
  jwksSchema,
  oauth2ClientListSchema,
  oauth2ClientSchema,
  rotateSecretResponseSchema,
  oauth2ScopeSchema,
  oauth2ScopeListSchema,
  introspectResponseSchema,
  revokeTokenResponseSchema,
  rotateKeysResponseSchema,
  oauth2GrantSchema,
} from "./oauth2";

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

describe("new OAuth2 schemas", () => {
  it("accepts client with rotation fields", () => {
    const payload = {
      client_id: "m2m-service",
      client_name: "M2M Service",
      grant_types: ["client_credentials"],
      response_types: [],
      redirect_uris: [],
      scopes: ["read:data"],
      is_public: false,
      previous_secret_expires_at: "2026-09-03T12:00:00Z",
      has_previous_secret: true,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-02T00:00:00Z",
    };
    expect(oauth2ClientSchema.safeParse(payload).success).toBe(true);
  });

  it("accepts rotateSecretResponseSchema", () => {
    const payload = {
      client_id: "client-1",
      client_secret: "sec_new_12345",
      previous_secret_expires_at: "2026-09-03T20:00:00Z",
    };
    expect(rotateSecretResponseSchema.safeParse(payload).success).toBe(true);
  });

  it("accepts oauth2ScopeSchema and oauth2ScopeListSchema", () => {
    const scope = {
      name: "profile",
      description: "Access user profile",
      claims: ["name", "email"],
      created_at: "2026-01-01T00:00:00Z",
    };
    expect(oauth2ScopeSchema.safeParse(scope).success).toBe(true);
    expect(oauth2ScopeListSchema.safeParse([scope]).success).toBe(true);
  });

  it("accepts introspectResponseSchema and revokeTokenResponseSchema", () => {
    const introspectPayload = {
      active: true,
      sub: "usr-456",
      scope: "openid profile",
      client_id: "client-1",
      exp: 1770000000,
    };
    expect(introspectResponseSchema.safeParse(introspectPayload).success).toBe(true);
    expect(revokeTokenResponseSchema.safeParse({ status: "revoked" }).success).toBe(true);
  });

  it("accepts rotateKeysResponseSchema and oauth2GrantSchema", () => {
    const rotateKey = {
      status: "rotated",
      new_kid: "key-2026-09",
      active_keys_count: 2,
    };
    expect(rotateKeysResponseSchema.safeParse(rotateKey).success).toBe(true);


    const grant = {
      grant_id: "grnt-1",
      client_id: "client-1",
      subject: "usr-456",
      scopes: ["openid"],
      created_at: "2026-09-01T00:00:00Z",
    };
    expect(oauth2GrantSchema.safeParse(grant).success).toBe(true);
  });
});

