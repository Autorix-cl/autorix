import { describe, it, expect, vi, beforeEach } from "vitest";
import { NextRequest } from "next/server";
import { GET as listOperators, POST as createOperator } from "./route";
import { PATCH as updateOperatorStatus, DELETE as deleteOperator } from "./[id]/route";
import { getCurrentOperator } from "@/lib/auth/session";
import type { Operator } from "@/lib/auth/types";

vi.mock("@/lib/auth/session", () => ({
  getCurrentOperator: vi.fn(),
  SESSION_COOKIE_NAME: "autorix_session",
}));

describe("Operators BFF Proxy Routes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getCurrentOperator).mockResolvedValue({
      id: "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
      name: "Admin",
      email: "admin@autorix.io",
      role: "owner",
      is_local: true,
      is_active: true,
    } as Operator);
    global.fetch = vi.fn();
  });

  it("lists operators via proxy", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify([
          {
            id: "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
            email: "admin@autorix.io",
            name: "Admin Operator",
            role: "owner",
            is_local: true,
            is_active: true,
          },
        ]),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const res = await listOperators();
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json).toHaveLength(1);
    expect(json[0].email).toBe("admin@autorix.io");
    expect(json[0].is_active).toBe(true);
  });

  it("creates an operator via proxy", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22",
          email: "newop@autorix.io",
          name: "New Operator",
          role: "operator",
          is_local: true,
          is_active: true,
        }),
        { status: 201, headers: { "Content-Type": "application/json" } }
      )
    );

    const req = new NextRequest("http://localhost/api/operators", {
      method: "POST",
      body: JSON.stringify({
        email: "newop@autorix.io",
        name: "New Operator",
        role: "operator",
        password: "SecretPassword123!",
      }),
    });

    const res = await createOperator(req);
    expect(res.status).toBe(201);
    const json = await res.json();
    expect(json.email).toBe("newop@autorix.io");
  });

  it("updates operator active status via PATCH proxy", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: "c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33",
          email: "target@autorix.io",
          name: "Target Operator",
          role: "operator",
          is_local: true,
          is_active: false,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const req = new NextRequest("http://localhost/api/operators/c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33", {
      method: "PATCH",
      body: JSON.stringify({ is_active: false }),
    });

    const res = await updateOperatorStatus(req, {
      params: Promise.resolve({ id: "c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33" }),
    });

    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.is_active).toBe(false);
  });

  it("deletes operator via DELETE proxy", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(null, { status: 204 })
    );

    const req = new NextRequest("http://localhost/api/operators/d3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44", {
      method: "DELETE",
    });

    const res = await deleteOperator(req, {
      params: Promise.resolve({ id: "d3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44" }),
    });

    expect(res.status).toBe(204);
  });

  it("blocks operator role from creating, updating, or deleting operators (prevents privilege escalation)", async () => {
    vi.mocked(getCurrentOperator).mockResolvedValue({
      id: "op-low-priv",
      name: "Regular Operator",
      email: "op@autorix.io",
      role: "operator",
      is_local: true,
      is_active: true,
    } as Operator);

    const postRes = await createOperator(
      new NextRequest("http://localhost/api/operators", {
        method: "POST",
        body: JSON.stringify({ email: "escalate@autorix.io", name: "Escalate", role: "owner", password: "Password123!" }),
      }),
    );
    expect(postRes.status).toBe(403);

    const patchRes = await updateOperatorStatus(
      new NextRequest("http://localhost/api/operators/op-1", {
        method: "PATCH",
        body: JSON.stringify({ role: "owner" }),
      }),
      { params: Promise.resolve({ id: "op-1" }) },
    );
    expect(patchRes.status).toBe(403);

    const delRes = await deleteOperator(new NextRequest("http://localhost/api/operators/op-1", { method: "DELETE" }), {
      params: Promise.resolve({ id: "op-1" }),
    });
    expect(delRes.status).toBe(403);
    expect(global.fetch).not.toHaveBeenCalled();
  });
});
