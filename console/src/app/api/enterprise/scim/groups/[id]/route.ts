import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";
import { scimGroupSchema } from "@/lib/api/schemas/hermes";

type Params = { params: Promise<{ id: string }> };

export async function GET(req: NextRequest, { params }: Params) {
  const { id } = await params;
  return proxyRequest("hermes", `/scim/v2/Groups/${encodeURIComponent(id)}`, scimGroupSchema);
}

export async function PUT(req: NextRequest, { params }: Params) {
  const { id } = await params;
  try {
    const body = await req.json();
    return proxyRequest("hermes", `/scim/v2/Groups/${encodeURIComponent(id)}`, scimGroupSchema, {
      method: "PUT",
      headers: { "Content-Type": "application/scim+json" },
      body: JSON.stringify(body),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to update group";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}

export async function PATCH(req: NextRequest, { params }: Params) {
  const { id } = await params;
  try {
    const body = await req.json();
    return proxyRequest("hermes", `/scim/v2/Groups/${encodeURIComponent(id)}`, scimGroupSchema, {
      method: "PATCH",
      headers: { "Content-Type": "application/scim+json" },
      body: JSON.stringify(body),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to patch group";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}

export async function DELETE(req: NextRequest, { params }: Params) {
  const { id } = await params;
  return proxyRequest("hermes", `/scim/v2/Groups/${encodeURIComponent(id)}`, z.any(), {
    method: "DELETE",
  });
}
