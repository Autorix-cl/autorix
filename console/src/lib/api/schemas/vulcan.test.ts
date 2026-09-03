import { describe, expect, it } from "vitest";
import { apiKeyListSchema, apiKeySchema, attenuateResponseSchema, createKeyResponseSchema } from "./vulcan";

describe("apiKeySchema", () => {
  it("accepts a realistic api key payload", () => {
    const payload = {
      id: "k-1",
      key_prefix: "av_live",
      key_hint: "ab12",
      name: "prod key",
      owner_id: "system",
      scopes: ["read", "write"],
      state: "active",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    expect(apiKeySchema.safeParse(payload).success).toBe(true);
  });

  it("accepts a list response", () => {
    expect(apiKeyListSchema.safeParse([]).success).toBe(true);
  });

  it("rejects an api key missing required fields", () => {
    const payload = { id: "k-1", name: "prod key" };
    expect(apiKeySchema.safeParse(payload).success).toBe(false);
  });
});

describe("createKeyResponseSchema", () => {
  it("accepts a realistic create-key response", () => {
    const payload = {
      api_key: {
        id: "k-1",
        key_prefix: "av_live",
        key_hint: "ab12",
        name: "prod key",
        owner_id: "system",
        scopes: [],
        state: "active",
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
      raw_token: "av_live_abc123",
      macaroon: {
        location: "https://api.autorix.io",
        key_id: "k-1",
        caveats: [{ predicate: "time_before = 2026-08-17T00:00:00Z" }],
        signature: "deadbeef",
      },
    };
    expect(createKeyResponseSchema.safeParse(payload).success).toBe(true);
  });

  it("rejects a response missing raw_token", () => {
    const payload = {
      api_key: {
        id: "k-1",
        key_prefix: "av_live",
        key_hint: "ab12",
        name: "prod key",
        owner_id: "system",
        scopes: [],
        state: "active",
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
      macaroon: {
        location: "https://api.autorix.io",
        key_id: "k-1",
        caveats: [],
        signature: "deadbeef",
      },
    };
    expect(createKeyResponseSchema.safeParse(payload).success).toBe(false);
  });
});

describe("attenuateResponseSchema", () => {
  it("accepts a bare macaroon", () => {
    const payload = {
      location: "https://api.autorix.io",
      key_id: "k-1",
      caveats: [{ predicate: "ip = 192.168.1.1" }],
      signature: "deadbeef",
    };
    expect(attenuateResponseSchema.safeParse(payload).success).toBe(true);
  });

  it("rejects a macaroon missing signature", () => {
    const payload = { location: "https://api.autorix.io", key_id: "k-1", caveats: [] };
    expect(attenuateResponseSchema.safeParse(payload).success).toBe(false);
  });
});

describe("Vulcan Engine Depth Schemas", () => {
  it("validates APIKey with usage tracking fields", () => {
    const payload = {
      id: "k-1",
      key_prefix: "av_live",
      key_hint: "ab12",
      name: "prod key",
      description: "Production microservice token",
      owner_id: "system",
      scopes: ["read", "write"],
      call_count: 42,
      last_source_ip: "192.168.1.100",
      grace_period_expires_at: "2026-09-04T00:00:00Z",
      state: "active",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    expect(apiKeySchema.safeParse(payload).success).toBe(true);
  });

  it("validates rotateKeyResponseSchema", async () => {
    const { rotateKeyResponseSchema } = await import("./vulcan");
    const payload = {
      api_key: {
        id: "k-1",
        key_prefix: "av_live",
        key_hint: "ab12",
        name: "prod key",
        owner_id: "system",
        scopes: [],
        state: "active",
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
      raw_token: "av_live_rotated_123",
      macaroon: {
        location: "https://api.autorix.io",
        key_id: "k-1",
        caveats: [],
        signature: "deadbeef",
      },
    };
    expect(rotateKeyResponseSchema.safeParse(payload).success).toBe(true);
  });

  it("validates scopeSchema and scopeListSchema", async () => {
    const { scopeSchema, scopeListSchema } = await import("./vulcan");
    const scope = {
      name: "auth:admin",
      description: "Full administrative access",
      created_at: "2026-09-02T10:00:00Z",
    };
    expect(scopeSchema.safeParse(scope).success).toBe(true);
    expect(scopeListSchema.safeParse([scope]).success).toBe(true);
  });

  it("validates verifyKeyResponseSchema", async () => {
    const { verifyKeyResponseSchema } = await import("./vulcan");
    expect(verifyKeyResponseSchema.safeParse({ valid: true }).success).toBe(true);
    expect(verifyKeyResponseSchema.safeParse({ valid: false, error: "expired caveat" }).success).toBe(true);
  });
});

