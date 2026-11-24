import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { GET, POST } from "./route";

vi.mock("@/lib/auth/session", () => ({
  getCurrentOperator: vi.fn().mockResolvedValue({
    id: "op-1",
    email: "admin@autorix.io",
    name: "Admin",
    role: "admin",
    is_local: true,
    is_active: true,
  }),
  SESSION_COOKIE_NAME: "autorix_session",
}));

describe("Vulcan API Routes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("POST /api/vulcan/keys proxies to Vulcan and returns metadata + one-time secret", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            api_key: {
              id: "k-1",
              name: "Test Key",
              key_prefix: "av_live_",
              key_hint: "abcd",
              scopes: ["read:users"],
              state: "active",
              created_at: "2026-01-01T00:00:00Z",
              expires_at: "2026-01-31T00:00:00Z",
            },
            raw_token: "av_live_secret_token_once",
            macaroon: { location: "https://api.autorix.io", key_id: "k-1", caveats: [], signature: "sig" },
          }),
          { status: 201, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );

    const req = new Request("http://localhost/api/vulcan/keys", {
      method: "POST",
      body: JSON.stringify({
        name: "Test Key",
        expires_in: "30d",
        scopes: ["read:users"],
      }),
    });

    const res = await POST(req);
    expect(res.status).toBe(201);

    const body = await res.json();
    expect(body.metadata).toBeDefined();
    expect(body.metadata.name).toBe("Test Key");
    expect(body.metadata.prefix).toBe("av_live_abcd");
    expect(body.secret).toBe("av_live_secret_token_once");
    expect(body.metadata.secret).toBeUndefined();
  });

  it("POST /api/vulcan/keys propagates upstream failure instead of fabricating a local 201 key", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValueOnce(new TypeError("connect ECONNREFUSED")));

    const req = new Request("http://localhost/api/vulcan/keys", {
      method: "POST",
      body: JSON.stringify({
        name: "Offline Key",
        expires_in: "30d",
        scopes: ["read:users"],
      }),
    });

    const res = await POST(req);
    expect(res.status).toBe(502);
  });

  it("GET /api/vulcan/keys strips any secret fields from returned list", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(
          JSON.stringify([
            {
              id: "k-1",
              name: "Test Key",
              key_prefix: "av_live_",
              key_hint: "abcd",
              secret: "should-never-leak",
              root_signature_key: "should-never-leak",
            },
          ]),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );

    const req = new Request("http://localhost/api/vulcan/keys", { method: "GET" });
    const res = await GET(req);

    expect(res.status).toBe(200);
    const keys = await res.json();
    expect(keys).toHaveLength(1);
    expect(keys[0].name).toBe("Test Key");
    expect(keys[0].secret).toBeUndefined();
    expect(keys[0].root_signature_key).toBeUndefined();
  });

  it("POST /api/vulcan/keys rejects invalid payload with 400", async () => {
    const req = new Request("http://localhost/api/vulcan/keys", {
      method: "POST",
      body: JSON.stringify({ name: "a" }),
    });

    const res = await POST(req);
    expect(res.status).toBe(400);

    const body = await res.json();
    expect(body.error).toBe("Invalid input");
  });
});
