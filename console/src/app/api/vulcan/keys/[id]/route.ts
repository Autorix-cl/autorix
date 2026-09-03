import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";
import { vulcanDb } from "@/lib/server/db";

type Params = { params: Promise<{ id: string }> };

// GET /api/vulcan/keys/[id]
export async function GET(req: NextRequest, { params }: Params) {
  const { id } = await params;
  const res = await proxyRequest("vulcan", `/admin/keys/${id}`, z.any());
  if (res.ok) return res;

  const localKey = vulcanDb.getKeyById(id);
  if (localKey) return NextResponse.json(localKey);

  return res;
}

// PATCH /api/vulcan/keys/[id]
export async function PATCH(req: NextRequest, { params }: Params) {
  const { id } = await params;
  try {
    const body = await req.json();
    const res = await proxyRequest("vulcan", `/admin/keys/${id}`, z.any(), {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (res.ok) return res;

    const localKey = vulcanDb.getKeyById(id);
    if (localKey) {
      if (body.name) localKey.name = body.name;
      if (body.scopes) localKey.scopes = body.scopes;
      return NextResponse.json(localKey);
    }

    return res;
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to update key";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}


// DELETE /api/vulcan/keys/[id] - Revoke a key
export async function DELETE(
  req: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const resolvedParams = await params;
  const id = resolvedParams.id;

  if (!id) {
    return NextResponse.json({ error: "Missing ID" }, { status: 400 });
  }

  // Attempt Go proxy revocation
  const res = await proxyRequest("vulcan", `/keys/${id}`, z.any(), { method: "DELETE" });
  if (res.ok) {
    return res;
  }

  // Fallback to local DB
  const success = vulcanDb.revokeKey(id);
  if (!success) {
    return NextResponse.json({ error: "Key not found" }, { status: 404 });
  }

  return NextResponse.json({ success: true }, { status: 200 });
}
