import { describe, it, expect, vi, afterEach } from "vitest";
import { NextRequest } from "next/server";
import { GET } from "./route";

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

describe("BFF Audit Verify Route (P8-S1-T5)", () => {
  it("proxies GET /api/audit/verify to Argus /admin/audit/verify", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          verified: true,
          chain_length: 100,
          head_hash: "abcd",
          verified_at: "2026-08-18T20:00:00Z",
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const req = new NextRequest("http://localhost:3000/api/audit/verify");
    const res = await GET(req);

    expect(res.status).toBe(200);
    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toContain("/admin/audit/verify");
    expect(init.method).toBe("GET");
  });
});
