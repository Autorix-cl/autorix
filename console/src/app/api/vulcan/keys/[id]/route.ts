import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";

type Params = { params: Promise<{ id: string }> };

function stripKeySecrets(record: unknown): unknown {
  if (!record || typeof record !== "object" || Array.isArray(record)) return record;
  const sanitized = { ...(record as Record<string, unknown>) };
  delete sanitized.secret;
  delete sanitized.raw_token;
  delete sanitized.root_signature_key;
  delete sanitized.prev_root_signature_key;
  delete sanitized.key_hash;
  return sanitized;
}

// GET /api/vulcan/keys/[id]
export async function GET(_req: NextRequest, { params }: Params) {
  const { id } = await params;
  const res = await proxyRequest("vulcan", `/admin/keys/${encodeURIComponent(id)}`, z.any());
  if (!res.ok) return res;
  const data = await res.json();
  return NextResponse.json(stripKeySecrets(data), { status: res.status });
}

// PATCH /api/vulcan/keys/[id]
export async function PATCH(req: NextRequest, { params }: Params) {
  const { id } = await params;
  try {
    const body = await req.json();
    const res = await proxyRequest("vulcan", `/admin/keys/${encodeURIComponent(id)}`, z.any(), {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!res.ok) return res;
    const data = await res.json();
    return NextResponse.json(stripKeySecrets(data), { status: res.status });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to update key";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}

// DELETE /api/vulcan/keys/[id] - Revoke a key
export async function DELETE(_req: Request, { params }: Params) {
  const { id } = await params;

  if (!id) {
    return NextResponse.json({ error: "Missing ID" }, { status: 400 });
  }

  return proxyRequest("vulcan", `/keys/${encodeURIComponent(id)}`, z.any(), { method: "DELETE" });
}
