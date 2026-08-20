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
