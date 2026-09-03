import { describe, it, expect, vi, beforeEach } from "vitest";
import { NextRequest, NextResponse } from "next/server";

vi.mock("@/lib/api/proxy", () => ({
  proxyRequest: vi.fn(),
}));

import { proxyRequest } from "@/lib/api/proxy";

describe("Nexus BFF Routes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("POST /api/nexus/check", () => {
    it("proxies check request to nexus engine with explain", async () => {
      const { POST } = await import("./check/route");
      vi.mocked(proxyRequest).mockResolvedValue(
        NextResponse.json({
          allowed: true,
          reason: "matched",
          trace: {
            namespace: "document",
            object: "doc_1",
            relation: "viewer",
            allowed: true,
          },
        })
      );

      const req = new NextRequest("http://localhost/api/nexus/check", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          subject: "user:alice",
          relation: "viewer",
          object: "document:doc_1",
          explain: true,
        }),
      });
      const res = await POST(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "nexus",
        "/check",
        expect.anything(),
        expect.objectContaining({
          method: "POST",
          body: expect.stringContaining('"namespace":"document"'),
        })
      );
      expect(res.status).toBe(200);
    });

    it("rejects invalid request missing parameters", async () => {
      const { POST } = await import("./check/route");
      const req = new NextRequest("http://localhost/api/nexus/check", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      const res = await POST(req);
      expect(res.status).toBe(400);
    });
  });

  describe("GET, POST, DELETE /api/nexus/tuples", () => {
    it("proxies GET /api/nexus/tuples with query parameters", async () => {
      const { GET } = await import("./tuples/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json([]));

      const req = new NextRequest("http://localhost/api/nexus/tuples?namespace=doc&limit=25");
      const res = await GET(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "nexus",
        "/tuples?namespace=doc&limit=25",
        expect.anything()
      );
      expect(res.status).toBe(200);
    });

    it("proxies POST /api/nexus/tuples to write tuples", async () => {
      const { POST } = await import("./tuples/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json([{ namespace: "doc", object: "1" }]));

      const req = new NextRequest("http://localhost/api/nexus/tuples", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          namespace: "document",
          object: "doc_1",
          relation: "viewer",
          subject_namespace: "user",
          subject_id: "alice",
        }),
      });
      const res = await POST(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "nexus",
        "/tuples",
        expect.anything(),
        expect.objectContaining({ method: "POST" })
      );
      expect(res.status).toBe(200);
    });

    it("proxies DELETE /api/nexus/tuples to delete tuples", async () => {
      const { DELETE } = await import("./tuples/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ status: "deleted" }));

      const req = new NextRequest("http://localhost/api/nexus/tuples?ids=document:doc_1#viewer@user:alice", {
        method: "DELETE",
      });
      const res = await DELETE(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "nexus",
        "/tuples",
        expect.anything(),
        expect.objectContaining({ method: "DELETE" })
      );
      expect(res.status).toBe(200);
    });
  });

  describe("POST /api/nexus/expand", () => {
    it("proxies expand request to nexus /expand", async () => {
      const { POST } = await import("./expand/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ tree: { type: "leaf" } }));

      const req = new NextRequest("http://localhost/api/nexus/expand", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          namespace: "document",
          object: "doc_1",
          relation: "viewer",
        }),
      });
      const res = await POST(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "nexus",
        "/expand",
        expect.anything(),
        expect.objectContaining({ method: "POST" })
      );
      expect(res.status).toBe(200);
    });
  });

  describe("POST /api/nexus/lookup", () => {
    it("proxies lookup subjects", async () => {
      const { POST } = await import("./lookup/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ subjects: [] }));

      const req = new NextRequest("http://localhost/api/nexus/lookup", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          type: "subjects",
          namespace: "document",
          object: "doc_1",
          relation: "viewer",
        }),
      });
      const res = await POST(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "nexus",
        "/lookup/subjects",
        expect.anything(),
        expect.objectContaining({ method: "POST" })
      );
      expect(res.status).toBe(200);
    });

    it("proxies lookup resources", async () => {
      const { POST } = await import("./lookup/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ resources: ["doc_1"] }));

      const req = new NextRequest("http://localhost/api/nexus/lookup", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          type: "resources",
          namespace: "document",
          relation: "viewer",
          subject_id: "alice",
        }),
      });
      const res = await POST(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "nexus",
        "/lookup/resources",
        expect.anything(),
        expect.objectContaining({ method: "POST" })
      );
      expect(res.status).toBe(200);
    });
  });

  describe("GET, POST, DELETE /api/nexus/caveats", () => {
    it("proxies GET /api/nexus/caveats", async () => {
      const { GET } = await import("./caveats/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json([]));

      const res = await GET();

      expect(proxyRequest).toHaveBeenCalledWith("nexus", "/admin/caveats", expect.anything());
      expect(res.status).toBe(200);
    });

    it("proxies POST /api/nexus/caveats", async () => {
      const { POST } = await import("./caveats/route");
      vi.mocked(proxyRequest).mockResolvedValue(
        NextResponse.json({ name: "is_admin", cel_expression: "true" })
      );

      const req = new NextRequest("http://localhost/api/nexus/caveats", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: "is_admin", cel_expression: "true" }),
      });
      const res = await POST(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "nexus",
        "/admin/caveats",
        expect.anything(),
        expect.objectContaining({ method: "POST" })
      );
      expect(res.status).toBe(200);
    });

    it("proxies DELETE /api/nexus/caveats?name=is_admin", async () => {
      const { DELETE } = await import("./caveats/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ status: "deleted" }));

      const req = new NextRequest("http://localhost/api/nexus/caveats?name=is_admin", {
        method: "DELETE",
      });
      const res = await DELETE(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "nexus",
        "/admin/caveats/is_admin",
        expect.anything(),
        expect.objectContaining({ method: "DELETE" })
      );
      expect(res.status).toBe(200);
    });
  });
});
