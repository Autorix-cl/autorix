import { describe, it, expect, vi, afterEach } from "vitest";
import { NextRequest } from "next/server";
import { GET, POST } from "./route";

vi.mock("@/lib/auth/session", () => ({
  getCurrentOperator: vi.fn().mockResolvedValue({
    id: "op_1",
    email: "admin@autorix.io",
    role: "admin",
    name: "Admin",
  }),
  getSessionToken: vi.fn().mockResolvedValue("mock-token"),
}));

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("BFF Audit Routes (P8-S1)", () => {
  it("proxies GET /api/audit to Argus /admin/audit with query parameters", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: "aud_1", action: "CREATE" }]), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const req = new NextRequest("http://localhost:3000/api/audit?actor=admin&limit=50");
    const res = await GET(req);

    expect(res.status).toBe(200);
    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toContain("/admin/audit?actor=admin&limit=50");
    expect(init.method).toBe("GET");
  });

  it("proxies POST /api/audit to Argus /admin/audit with body", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: "aud_1", status: "recorded" }), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const auditPayload = {
      action: "UPDATE",
      resource_type: "policy",
      resource_id: "pol_1",
      actor: "admin@enterprise.io",
      outcome: "success",
    };

    const req = new NextRequest("http://localhost:3000/api/audit", {
      method: "POST",
      body: JSON.stringify(auditPayload),
      headers: { "Content-Type": "application/json" },
    });

    const res = await POST(req);

    expect(res.status).toBe(201);
    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toContain("/admin/audit");
    expect(init.method).toBe("POST");
    expect(init.body).toBe(JSON.stringify(auditPayload));
  });
});
