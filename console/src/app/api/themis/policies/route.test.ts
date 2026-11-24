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

describe("Themis Policies API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("POST /api/themis/policies forwards valid CEL expressions including CEL macros directly to Themis", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: "pol-1",
          name: "Prod Guardrail",
          expression: 'request.tags.exists(t, t == "prod") && "admin" in request.roles',
          priority: 10,
          enabled: true,
        }),
        { status: 201, headers: { "Content-Type": "application/json" } },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const req = new Request("http://localhost/api/themis/policies", {
      method: "POST",
      body: JSON.stringify({
        name: "Prod Guardrail",
        expression: 'request.tags.exists(t, t == "prod") && "admin" in request.roles',
        priority: 10,
      }),
    });

    const res = await POST(req);
    expect(res.status).toBe(201);
    const body = await res.json();
    expect(body.id).toBe("pol-1");
    expect(body.name).toBe("Prod Guardrail");
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("POST /api/themis/policies propagates upstream validation/unreachable errors instead of fabricating 201", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "CEL syntax error at column 14" }), {
          status: 400,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );

    const req = new Request("http://localhost/api/themis/policies", {
      method: "POST",
      body: JSON.stringify({
        name: "Bad Policy",
        expression: 'request.role =="',
        priority: 10,
      }),
    });

    const res = await POST(req);
    expect(res.status).toBe(400);
    const body = await res.json();
    expect(body.error).toContain("CEL syntax error");

    vi.stubGlobal("fetch", vi.fn().mockRejectedValueOnce(new TypeError("connect ECONNREFUSED")));
    const offlineRes = await POST(
      new Request("http://localhost/api/themis/policies", {
        method: "POST",
        body: JSON.stringify({ name: "Offline Policy", expression: "true", priority: 1 }),
      }),
    );
    expect(offlineRes.status).toBe(502);
  });

  it("GET /api/themis/policies proxies list from Themis engine", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(
          JSON.stringify([
            { id: "p1", name: "P1", expression: "true", priority: 10, enabled: true },
            { id: "p2", name: "P2", expression: "true", priority: 20, enabled: true },
          ]),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );

    const req = new Request("http://localhost/api/themis/policies", { method: "GET" });
    const res = await GET(req);

    expect(res.status).toBe(200);
    const policies = await res.json();
    expect(policies).toHaveLength(2);
    expect(policies[0].name).toBe("P1");
    expect(policies[1].name).toBe("P2");
  });
});
