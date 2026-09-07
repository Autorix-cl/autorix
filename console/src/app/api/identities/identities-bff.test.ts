import { describe, it, expect, vi, beforeEach } from "vitest";
import { NextRequest, NextResponse } from "next/server";

vi.mock("@/lib/api/proxy", () => ({
  proxyRequest: vi.fn(),
}));

import { proxyRequest } from "@/lib/api/proxy";

describe("Ego Identities BFF Routes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("GET, PATCH, DELETE /api/identities/[id]", () => {
    it("proxies GET /api/identities/[id]", async () => {
      const { GET } = await import("./[id]/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ id: "id-1", state: "active" }));

      const req = new NextRequest("http://localhost/api/identities/id-1");
      const res = await GET(req, { params: Promise.resolve({ id: "id-1" }) });

      expect(proxyRequest).toHaveBeenCalledWith("ego", "/admin/identities/id-1", expect.anything());
      expect(res.status).toBe(200);
    });

    it("proxies PATCH /api/identities/[id]", async () => {
      const { PATCH } = await import("./[id]/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ id: "id-1", state: "suspended" }));

      const req = new NextRequest("http://localhost/api/identities/id-1", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ state: "suspended" }),
      });
      const res = await PATCH(req, { params: Promise.resolve({ id: "id-1" }) });

      expect(proxyRequest).toHaveBeenCalledWith(
        "ego",
        "/admin/identities/id-1",
        expect.anything(),
        expect.objectContaining({ method: "PATCH" }),
      );
      expect(res.status).toBe(200);
    });

    it("proxies DELETE /api/identities/[id]", async () => {
      const { DELETE } = await import("./[id]/route");
      vi.mocked(proxyRequest).mockResolvedValue(new NextResponse(null, { status: 204 }));

      const req = new NextRequest("http://localhost/api/identities/id-1", { method: "DELETE" });
      const res = await DELETE(req, { params: Promise.resolve({ id: "id-1" }) });

      expect(proxyRequest).toHaveBeenCalledWith(
        "ego",
        "/admin/identities/id-1",
        expect.anything(),
        expect.objectContaining({ method: "DELETE" }),
      );
      expect(res.status).toBe(204);
    });
  });

  describe("GET, DELETE /api/identities/[id]/sessions", () => {
    it("proxies GET identity sessions", async () => {
      const { GET } = await import("./[id]/sessions/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json([]));

      const req = new NextRequest("http://localhost/api/identities/id-1/sessions");
      const res = await GET(req, { params: Promise.resolve({ id: "id-1" }) });

      expect(proxyRequest).toHaveBeenCalledWith("ego", "/admin/identities/id-1/sessions", expect.anything());
      expect(res.status).toBe(200);
    });

    it("proxies DELETE identity sessions", async () => {
      const { DELETE } = await import("./[id]/sessions/route");
      vi.mocked(proxyRequest).mockResolvedValue(new NextResponse(null, { status: 204 }));

      const req = new NextRequest("http://localhost/api/identities/id-1/sessions", { method: "DELETE" });
      const res = await DELETE(req, { params: Promise.resolve({ id: "id-1" }) });

      expect(proxyRequest).toHaveBeenCalledWith(
        "ego",
        "/admin/identities/id-1/sessions",
        expect.anything(),
        expect.objectContaining({ method: "DELETE" }),
      );
      expect(res.status).toBe(204);
    });
  });

  describe("GET /api/sessions & DELETE /api/sessions/[id]", () => {
    it("proxies GET global sessions with pagination", async () => {
      const { GET } = await import("../sessions/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ data: [] }));

      const req = new NextRequest("http://localhost/api/sessions?limit=20");
      const res = await GET(req);

      expect(proxyRequest).toHaveBeenCalledWith("ego", "/admin/sessions?limit=20", expect.anything());
      expect(res.status).toBe(200);
    });

    it("proxies DELETE session by ID", async () => {
      const { DELETE } = await import("../sessions/[id]/route");
      vi.mocked(proxyRequest).mockResolvedValue(new NextResponse(null, { status: 204 }));

      const req = new NextRequest("http://localhost/api/sessions/sess-1", { method: "DELETE" });
      const res = await DELETE(req, { params: Promise.resolve({ id: "sess-1" }) });

      expect(proxyRequest).toHaveBeenCalledWith(
        "ego",
        "/admin/sessions/sess-1",
        expect.anything(),
        expect.objectContaining({ method: "DELETE" }),
      );
      expect(res.status).toBe(204);
    });
  });

  describe("Credentials, Reset Password, Recovery Link, MFA", () => {
    it("proxies GET credentials list", async () => {
      const { GET } = await import("./[id]/credentials/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json([]));

      const req = new NextRequest("http://localhost/api/identities/id-1/credentials");
      const res = await GET(req, { params: Promise.resolve({ id: "id-1" }) });

      expect(proxyRequest).toHaveBeenCalledWith("ego", "/admin/identities/id-1/credentials", expect.anything());
      expect(res.status).toBe(200);
    });

    it("proxies POST reset password", async () => {
      const { POST } = await import("./[id]/credentials/reset-password/route");
      vi.mocked(proxyRequest).mockResolvedValue(
        NextResponse.json({ status: "password_reset", temporary_password: "abc", force_rotation: true }),
      );

      const req = new NextRequest("http://localhost/api/identities/id-1/credentials/reset-password", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ force_rotation: true }),
      });
      const res = await POST(req, { params: Promise.resolve({ id: "id-1" }) });

      expect(proxyRequest).toHaveBeenCalledWith(
        "ego",
        "/admin/identities/id-1/credentials/reset-password",
        expect.anything(),
        expect.objectContaining({ method: "POST" }),
      );
      expect(res.status).toBe(200);
    });

    it("proxies POST recovery link", async () => {
      const { POST } = await import("./[id]/recovery-link/route");
      vi.mocked(proxyRequest).mockResolvedValue(
        NextResponse.json({ recovery_link: "/recovery?token=123", token: "123", expires_at: "2026-01-01" }),
      );

      const req = new NextRequest("http://localhost/api/identities/id-1/recovery-link", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ expires_in: "2h" }),
      });
      const res = await POST(req, { params: Promise.resolve({ id: "id-1" }) });

      expect(proxyRequest).toHaveBeenCalledWith(
        "ego",
        "/admin/identities/id-1/recovery-link",
        expect.anything(),
        expect.objectContaining({ method: "POST" }),
      );
      expect(res.status).toBe(200);
    });

    it("proxies GET and DELETE MFA", async () => {
      const { GET, DELETE } = await import("./[id]/mfa/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ totp_enabled: true }));

      const getReq = new NextRequest("http://localhost/api/identities/id-1/mfa");
      const getRes = await GET(getReq, { params: Promise.resolve({ id: "id-1" }) });
      expect(proxyRequest).toHaveBeenCalledWith("ego", "/admin/identities/id-1/mfa", expect.anything());
      expect(getRes.status).toBe(200);

      vi.mocked(proxyRequest).mockResolvedValue(new NextResponse(null, { status: 204 }));
      const delReq = new NextRequest("http://localhost/api/identities/id-1/mfa", { method: "DELETE" });
      const delRes = await DELETE(delReq, { params: Promise.resolve({ id: "id-1" }) });
      expect(proxyRequest).toHaveBeenCalledWith(
        "ego",
        "/admin/identities/id-1/mfa",
        expect.anything(),
        expect.objectContaining({ method: "DELETE" }),
      );
      expect(delRes.status).toBe(204);
    });
  });

  describe("POST /api/identities/bulk-import", () => {
    it("performs dry run validation without calling backend", async () => {
      const { POST } = await import("./bulk-import/route");
      const req = new NextRequest("http://localhost/api/identities/bulk-import", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          rows: [
            { email: "alice@autorix.io", firstName: "Alice" },
            { email: "invalid-email", firstName: "Bob" },
          ],
          dry_run: true,
        }),
      });
      const res = await POST(req);
      const data = await res.json();

      expect(proxyRequest).not.toHaveBeenCalled();
      expect(data.total).toBe(2);
      expect(data.succeeded).toBe(1);
      expect(data.failed).toBe(1);
      expect(data.errors).toHaveLength(1);
      expect(data.dry_run).toBe(true);
    });

    it("commits valid rows when dry_run is false", async () => {
      const { POST } = await import("./bulk-import/route");
      vi.mocked(proxyRequest).mockResolvedValue(
        NextResponse.json({ identity: { id: "new-id" }, session: { id: "new-sess" } }),
      );

      const req = new NextRequest("http://localhost/api/identities/bulk-import", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          rows: [{ email: "alice@autorix.io", firstName: "Alice" }],
          dry_run: false,
        }),
      });
      const res = await POST(req);
      const data = await res.json();

      expect(proxyRequest).toHaveBeenCalledTimes(1);
      expect(data.succeeded).toBe(1);
      expect(data.failed).toBe(0);
    });
  });

  describe("GET, POST /api/identities/schemas & GET, PATCH, DELETE /api/identities/schemas/[id]", () => {
    it("proxies GET /api/identities/schemas", async () => {
      const { GET } = await import("./schemas/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json([{ id: "default", name: "User Identity" }]));

      const res = await GET();
      expect(proxyRequest).toHaveBeenCalledWith("ego", "/admin/schemas", expect.anything());
      expect(res.status).toBe(200);
    });

    it("proxies POST /api/identities/schemas", async () => {
      const { POST } = await import("./schemas/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ id: "partner_v1" }, { status: 201 }));

      const req = new NextRequest("http://localhost/api/identities/schemas", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          id: "partner_v1",
          name: "Partner Schema",
          schema: { type: "object" },
        }),
      });
      const res = await POST(req);
      expect(proxyRequest).toHaveBeenCalledWith(
        "ego",
        "/admin/schemas",
        expect.anything(),
        expect.objectContaining({ method: "POST" }),
      );
      expect(res.status).toBe(201);
    });

    it("proxies GET /api/identities/schemas/[id]", async () => {
      const { GET } = await import("./schemas/[id]/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ id: "partner_v1" }));

      const req = new NextRequest("http://localhost/api/identities/schemas/partner_v1");
      const res = await GET(req, { params: Promise.resolve({ id: "partner_v1" }) });
      expect(proxyRequest).toHaveBeenCalledWith("ego", "/admin/schemas/partner_v1", expect.anything());
      expect(res.status).toBe(200);
    });

    it("proxies PATCH /api/identities/schemas/[id]", async () => {
      const { PATCH } = await import("./schemas/[id]/route");
      vi.mocked(proxyRequest).mockResolvedValue(NextResponse.json({ id: "partner_v1", name: "Updated" }));

      const req = new NextRequest("http://localhost/api/identities/schemas/partner_v1", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: "Updated" }),
      });
      const res = await PATCH(req, { params: Promise.resolve({ id: "partner_v1" }) });
      expect(proxyRequest).toHaveBeenCalledWith(
        "ego",
        "/admin/schemas/partner_v1",
        expect.anything(),
        expect.objectContaining({ method: "PATCH" }),
      );
      expect(res.status).toBe(200);
    });

    it("proxies DELETE /api/identities/schemas/[id]", async () => {
      const { DELETE } = await import("./schemas/[id]/route");
      vi.mocked(proxyRequest).mockResolvedValue(new NextResponse(null, { status: 204 }));

      const req = new NextRequest("http://localhost/api/identities/schemas/partner_v1", {
        method: "DELETE",
      });
      const res = await DELETE(req, { params: Promise.resolve({ id: "partner_v1" }) });
      expect(proxyRequest).toHaveBeenCalledWith(
        "ego",
        "/admin/schemas/partner_v1",
        expect.anything(),
        expect.objectContaining({ method: "DELETE" }),
      );
      expect(res.status).toBe(204);
    });
  });
});
