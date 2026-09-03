/* eslint-disable */
import { describe, it, expect, beforeEach, vi } from "vitest";
import { GET, POST } from "./route";
import { vulcanDb } from "@/lib/server/db";

// Mock the DB so we don't write to the real file during tests
vi.mock("@/lib/server/db", () => {
  let store: any[] = [];
  return {
    vulcanDb: {
      getKeys: () => store,
      getKeyById: (id: string) => store.find(k => k.id === id),
      addKey: (k: any) => store.push(k),
      revokeKey: (id: string) => {
        const k = store.find(k => k.id === id);
        if (k) { k.revoked = true; return true; }
        return false;
      },
      // Helper for tests to reset state
      _reset: () => { store = []; }
    }
  };
});

describe("Vulcan API Routes", () => {
  beforeEach(() => {
    (vulcanDb as any)._reset();
  });

  it("POST /api/vulcan/keys should create a key and return the secret ONCE", async () => {
    const payload = {
      name: "Test Key",
      expires_in: "30d",
      scopes: ["read:users"]
    };

    const req = new Request("http://localhost/api/vulcan/keys", {
      method: "POST",
      body: JSON.stringify(payload)
    });

    const res = await POST(req);
    expect(res.status).toBe(201);
    
    const body = await res.json();
    
    // Should return both metadata and secret
    expect(body.metadata).toBeDefined();
    expect(body.metadata.name).toBe("Test Key");
    expect(body.metadata.prefix).toMatch(/^av_live_.*\\*\\*\\*\\*$/);
    expect(body.secret).toBeDefined();
    expect(body.secret.startsWith("av_live_")).toBe(true);
    
    // Make sure secret is NOT in metadata
    expect(body.metadata.secret).toBeUndefined();
  });

  it("GET /api/vulcan/keys should return list WITHOUT secrets", async () => {
    // First create a key
    const payload = {
      name: "Test Key",
      expires_in: "30d",
      scopes: ["read:users"]
    };
    await POST(new Request("http://localhost", { method: "POST", body: JSON.stringify(payload) }));

    // Now GET them
    const req = new Request("http://localhost/api/vulcan/keys", { method: "GET" });
    const res = await GET(req);
    
    expect(res.status).toBe(200);
    const keys = await res.json();
    
    expect(keys.length).toBe(1);
    expect(keys[0].name).toBe("Test Key");
    // CRITICAL: The secret must NEVER be leaked in the list
    expect(keys[0].secret).toBeUndefined();
  });

  it("POST /api/vulcan/keys should reject invalid payload", async () => {
    const req = new Request("http://localhost/api/vulcan/keys", {
      method: "POST",
      body: JSON.stringify({ name: "a" }) // name too short, missing scopes
    });

    const res = await POST(req);
    expect(res.status).toBe(400);
    
    const body = await res.json();
    expect(body.error).toBe("Invalid input");
  });
});
