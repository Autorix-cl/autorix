import { describe, it, expect, vi, beforeEach } from "vitest";
import { NextRequest } from "next/server";
import { GET as listKeys, POST as createKey } from "./keys/route";
import { GET as getKey, PATCH as updateKey, DELETE as deleteKey } from "./keys/[id]/route";
import { POST as rotateKey } from "./keys/[id]/rotate/route";
import { GET as listScopes, POST as createScope } from "./scopes/route";
import { DELETE as deleteScope } from "./scopes/[name]/route";
import { POST as verifyKey } from "./verify/route";

describe("Vulcan BFF Proxy Routes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn();
  });

  it("lists keys via proxy", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify([
          {
            id: "k-1",
            name: "Backend Service Key",
            key_prefix: "av_live_",
            key_hint: "1234",
            scopes: ["read", "write"],
            call_count: 10,
            state: "active",
            created_at: "2026-09-02T10:00:00Z",
            updated_at: "2026-09-02T10:00:00Z",
          },
        ]),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const res = await listKeys(new NextRequest("http://localhost/api/vulcan/keys"));
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json).toHaveLength(1);
    expect(json[0].name).toBe("Backend Service Key");
  });

  it("creates a key via proxy", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          api_key: {
            id: "k-new",
            key_prefix: "av_live_",
            key_hint: "abcd",
            name: "Worker Key",
            owner_id: "system",
            scopes: ["tasks:run"],
            state: "active",
            created_at: "2026-09-02T10:00:00Z",
            updated_at: "2026-09-02T10:00:00Z",
          },
          raw_token: "av_live_xyz_token",
          macaroon: {
            location: "https://api.autorix.io",
            key_id: "k-new",
            caveats: [],
            signature: "sig123",
          },
        }),
        { status: 201, headers: { "Content-Type": "application/json" } }
      )
    );

    const req = new NextRequest("http://localhost/api/vulcan/keys", {
      method: "POST",
      body: JSON.stringify({ name: "Worker Key", scopes: ["tasks:run"], expires_in: "30d" }),
    });
    const res = await createKey(req);
    expect(res.status).toBe(201);
    const json = await res.json();
    expect(json.metadata.name).toBe("Worker Key");
    expect(json.secret).toBe("av_live_xyz_token");
  });

  it("gets, updates, and revokes a key", async () => {
    // 1. GET key
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ id: "k-1", name: "Key 1", scopes: ["read"] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    const getRes = await getKey(new NextRequest("http://localhost/api/vulcan/keys/k-1"), {
      params: Promise.resolve({ id: "k-1" }),
    });
    expect(getRes.status).toBe(200);

    // 2. PATCH key
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ id: "k-1", name: "Renamed Key" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    const patchRes = await updateKey(
      new NextRequest("http://localhost/api/vulcan/keys/k-1", {
        method: "PATCH",
        body: JSON.stringify({ name: "Renamed Key" }),
      }),
      { params: Promise.resolve({ id: "k-1" }) }
    );
    expect(patchRes.status).toBe(200);

    // 3. DELETE key
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ status: "revoked" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    const delRes = await deleteKey(new NextRequest("http://localhost/api/vulcan/keys/k-1", { method: "DELETE" }), {
      params: Promise.resolve({ id: "k-1" }),
    });
    expect(delRes.status).toBe(200);
  });

  it("rotates an API key with grace period", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          api_key: {
            id: "k-1",
            key_prefix: "av_live_",
            key_hint: "9999",
            name: "Rotated Key",
            owner_id: "system",
            scopes: ["admin"],
            grace_period_expires_at: "2026-09-04T10:00:00Z",
            state: "active",
            created_at: "2026-09-02T10:00:00Z",
            updated_at: "2026-09-02T10:00:00Z",
          },
          raw_token: "av_live_rotated_new_token",
          macaroon: {
            location: "https://api.autorix.io",
            key_id: "k-1",
            caveats: [],
            signature: "sig456",
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const req = new NextRequest("http://localhost/api/vulcan/keys/k-1/rotate", {
      method: "POST",
      body: JSON.stringify({ grace_period: "48h" }),
    });
    const res = await rotateKey(req, { params: Promise.resolve({ id: "k-1" }) });
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.raw_token).toBe("av_live_rotated_new_token");
    expect(json.api_key.grace_period_expires_at).toBeDefined();
  });

  it("manages scope catalogue", async () => {
    // 1. GET scopes
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify([{ name: "billing:read", description: "Read billing info" }]), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    const listRes = await listScopes();
    expect(listRes.status).toBe(200);
    const scopes = await listRes.json();
    expect(scopes).toHaveLength(1);

    // 2. POST scope
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ name: "billing:write", description: "Manage billing" }), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      })
    );
    const postRes = await createScope(
      new NextRequest("http://localhost/api/vulcan/scopes", {
        method: "POST",
        body: JSON.stringify({ name: "billing:write", description: "Manage billing" }),
      })
    );
    expect(postRes.status).toBe(201);

    // 3. DELETE scope
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ status: "deleted" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    const delRes = await deleteScope(
      new NextRequest("http://localhost/api/vulcan/scopes/billing:write", { method: "DELETE" }),
      { params: Promise.resolve({ name: "billing:write" }) }
    );
    expect(delRes.status).toBe(200);
  });

  it("verifies macaroons", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ valid: true }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    const req = new NextRequest("http://localhost/api/vulcan/verify", {
      method: "POST",
      body: JSON.stringify({
        macaroon: {
          location: "https://api.autorix.io",
          key_id: "k-1",
          caveats: [{ predicate: "ip = 192.168.1.1" }],
          signature: "deadbeef",
        },
        context: {
          now: "2026-09-02T10:00:00Z",
          ip_address: "192.168.1.1",
          method: "GET",
          path: "/api/v1/orders",
        },
      }),
    });
    const res = await verifyKey(req);
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.valid).toBe(true);
  });
});
