import { describe, it, expect, vi, beforeEach } from "vitest";
import { NextRequest, NextResponse } from "next/server";

vi.mock("@/lib/api/proxy", () => ({
  proxyRequest: vi.fn(),
}));

import { proxyRequest } from "@/lib/api/proxy";

describe("Aegis BFF Routes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("GET /api/proxy-rules/[id]", () => {
    it("proxies GET to aegis /rules/{id}", async () => {
      const { GET } = await import("./[id]/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ id: "rule-123" }));

      const req = new NextRequest("http://localhost/api/proxy-rules/rule-123");
      const res = await GET(req, { params: Promise.resolve({ id: "rule-123" }) });

      expect(proxyRequest).toHaveBeenCalledWith(
        "aegis",
        "/rules/rule-123",
        expect.anything()
      );
      expect(res.status).toBe(200);
    });
  });

  describe("PUT /api/proxy-rules/reorder", () => {
    it("proxies PUT to aegis /rules/reorder with validated payload", async () => {
      const { PUT } = await import("./reorder/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ status: "reordered" }));

      const req = new NextRequest("http://localhost/api/proxy-rules/reorder", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ids: ["rule-2", "rule-1"] }),
      });
      const res = await PUT(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "aegis",
        "/rules/reorder",
        expect.anything(),
        expect.objectContaining({
          method: "PUT",
          body: JSON.stringify({ ids: ["rule-2", "rule-1"] }),
        })
      );
      expect(res.status).toBe(200);
    });

    it("rejects invalid reorder payload with 400", async () => {
      const { PUT } = await import("./reorder/route");
      const req = new NextRequest("http://localhost/api/proxy-rules/reorder", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ invalid: true }),
      });
      const res = await PUT(req);

      expect(res.status).toBe(400);
      expect(proxyRequest).not.toHaveBeenCalled();
    });
  });

  describe("GET /api/proxy-rules/handlers", () => {
    it("proxies GET to aegis /handlers", async () => {
      const { GET } = await import("./handlers/route");
      vi.mocked(proxyRequest).mockResolvedValue(
        NextResponse.json({ authenticators: [], authorizers: [], mutators: [] })
      );

      const res = await GET();

      expect(proxyRequest).toHaveBeenCalledWith(
        "aegis",
        "/handlers",
        expect.anything()
      );
      expect(res.status).toBe(200);
    });
  });

  describe("GET /api/proxy-rules/versions", () => {
    it("proxies GET to aegis /rules/versions", async () => {
      const { GET } = await import("./versions/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json([]));

      const res = await GET();

      expect(proxyRequest).toHaveBeenCalledWith(
        "aegis",
        "/rules/versions",
        expect.anything()
      );
      expect(res.status).toBe(200);
    });
  });

  describe("POST /api/proxy-rules/rollback/[version]", () => {
    it("proxies POST to aegis /rules/rollback/{version}", async () => {
      const { POST } = await import("./rollback/[version]/route");
      vi.mocked(proxyRequest).mockResolvedValue(
        NextResponse.json({ status: "rolled_back", version: 2 })
      );

      const req = new NextRequest("http://localhost/api/proxy-rules/rollback/2", {
        method: "POST",
      });
      const res = await POST(req, { params: Promise.resolve({ version: "2" }) });

      expect(proxyRequest).toHaveBeenCalledWith(
        "aegis",
        "/rules/rollback/2",
        expect.anything(),
        expect.objectContaining({ method: "POST" })
      );
      expect(res.status).toBe(200);
    });
  });
});
