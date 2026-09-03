import { describe, it, expect, vi, beforeEach } from "vitest";
import { NextRequest, NextResponse } from "next/server";

vi.mock("@/lib/api/proxy", () => ({
  proxyRequest: vi.fn(),
}));

import { proxyRequest } from "@/lib/api/proxy";

describe("OAuth2 BFF Routes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("GET /api/oauth2/clients/[id]", () => {
    it("proxies GET to janus /admin/clients/{id}", async () => {
      const { GET } = await import("./clients/[id]/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ client_id: "client-1" }));

      const req = new NextRequest("http://localhost/api/oauth2/clients/client-1");
      const res = await GET(req, { params: Promise.resolve({ id: "client-1" }) });

      expect(proxyRequest).toHaveBeenCalledWith(
        "janus",
        "/admin/clients/client-1",
        expect.anything()
      );
      expect(res.status).toBe(200);
    });
  });

  describe("PATCH /api/oauth2/clients/[id]", () => {
    it("proxies PATCH to janus /admin/clients/{id}", async () => {
      const { PATCH } = await import("./clients/[id]/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ client_id: "client-1", client_name: "Updated" }));

      const req = new NextRequest("http://localhost/api/oauth2/clients/client-1", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ client_name: "Updated" }),
      });
      const res = await PATCH(req, { params: Promise.resolve({ id: "client-1" }) });

      expect(proxyRequest).toHaveBeenCalledWith(
        "janus",
        "/admin/clients/client-1",
        expect.anything(),
        expect.objectContaining({ method: "PATCH" })
      );
      expect(res.status).toBe(200);
    });
  });

  describe("DELETE /api/oauth2/clients/[id]", () => {
    it("proxies DELETE to janus /admin/clients/{id}", async () => {
      const { DELETE } = await import("./clients/[id]/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ status: "deleted" }));

      const req = new NextRequest("http://localhost/api/oauth2/clients/client-1", { method: "DELETE" });
      const res = await DELETE(req, { params: Promise.resolve({ id: "client-1" }) });

      expect(proxyRequest).toHaveBeenCalledWith(
        "janus",
        "/admin/clients/client-1",
        expect.anything(),
        expect.objectContaining({ method: "DELETE" })
      );
      expect(res.status).toBe(200);
    });
  });

  describe("POST /api/oauth2/clients/[id]/rotate-secret", () => {
    it("proxies POST to janus /admin/clients/{id}/rotate-secret", async () => {
      const { POST } = await import("./clients/[id]/rotate-secret/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ client_secret: "new_sec" }));

      const req = new NextRequest("http://localhost/api/oauth2/clients/client-1/rotate-secret", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ overlap_seconds: 3600 }),
      });
      const res = await POST(req, { params: Promise.resolve({ id: "client-1" }) });

      expect(proxyRequest).toHaveBeenCalledWith(
        "janus",
        "/admin/clients/client-1/rotate-secret",
        expect.anything(),
        expect.objectContaining({ method: "POST" })
      );
      expect(res.status).toBe(200);
    });
  });

  describe("GET and POST /api/oauth2/scopes", () => {
    it("proxies GET to janus /admin/scopes", async () => {
      const { GET } = await import("./scopes/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json([{ name: "openid" }]));

      const req = new NextRequest("http://localhost/api/oauth2/scopes");
      const res = await GET(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "janus",
        "/admin/scopes",
        expect.anything()
      );
      expect(res.status).toBe(200);
    });

    it("proxies POST to janus /admin/scopes", async () => {
      const { POST } = await import("./scopes/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ name: "billing:read" }));

      const req = new NextRequest("http://localhost/api/oauth2/scopes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: "billing:read", description: "Billing read scope", claims: ["org_id"] }),
      });
      const res = await POST(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "janus",
        "/admin/scopes",
        expect.anything(),
        expect.objectContaining({ method: "POST" })
      );
      expect(res.status).toBe(200);
    });
  });

  describe("DELETE /api/oauth2/scopes/[name]", () => {
    it("proxies DELETE to janus /admin/scopes/{name}", async () => {
      const { DELETE } = await import("./scopes/[name]/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ status: "deleted" }));

      const req = new NextRequest("http://localhost/api/oauth2/scopes/custom:scope", { method: "DELETE" });
      const res = await DELETE(req, { params: Promise.resolve({ name: "custom:scope" }) });

      expect(proxyRequest).toHaveBeenCalledWith(
        "janus",
        "/admin/scopes/custom:scope",
        expect.anything(),
        expect.objectContaining({ method: "DELETE" })
      );
      expect(res.status).toBe(200);
    });
  });

  describe("POST /api/oauth2/tokens/introspect", () => {
    it("proxies POST to janus /oauth2/introspect with form urlencoded or json body", async () => {
      const { POST } = await import("./tokens/introspect/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ active: true }));

      const req = new NextRequest("http://localhost/api/oauth2/tokens/introspect", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token: "header.payload.signature" }),
      });
      const res = await POST(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "janus",
        "/oauth2/introspect",
        expect.anything(),
        expect.objectContaining({ method: "POST" })
      );
      expect(res.status).toBe(200);
    });
  });

  describe("POST /api/oauth2/tokens/revoke", () => {
    it("proxies POST to janus /oauth2/revoke", async () => {
      const { POST } = await import("./tokens/revoke/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ status: "revoked" }));

      const req = new NextRequest("http://localhost/api/oauth2/tokens/revoke", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token: "header.payload.signature" }),
      });
      const res = await POST(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "janus",
        "/oauth2/revoke",
        expect.anything(),
        expect.objectContaining({ method: "POST" })
      );
      expect(res.status).toBe(200);
    });
  });

  describe("POST /api/oauth2/keys/rotate", () => {
    it("proxies POST to janus /admin/keys/rotate", async () => {
      const { POST } = await import("./keys/rotate/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ status: "rotated", new_kid: "key-1" }));

      const req = new NextRequest("http://localhost/api/oauth2/keys/rotate", { method: "POST" });
      const res = await POST(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "janus",
        "/admin/keys/rotate",
        expect.anything(),
        expect.objectContaining({ method: "POST" })
      );
      expect(res.status).toBe(200);
    });
  });

  describe("GET /api/oauth2/grants", () => {
    it("proxies GET to janus /admin/grants", async () => {
      const { GET } = await import("./grants/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json([]));

      const req = new NextRequest("http://localhost/api/oauth2/grants");
      const res = await GET(req);

      expect(proxyRequest).toHaveBeenCalledWith(
        "janus",
        "/admin/grants",
        expect.anything()
      );
      expect(res.status).toBe(200);
    });
  });
});
