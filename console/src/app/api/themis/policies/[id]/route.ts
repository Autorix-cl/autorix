import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";

type Params = { params: Promise<{ id: string }> };

export async function GET(req: NextRequest, { params }: Params) {
  const { id } = await params;
  const tenantId = req.nextUrl.searchParams.get("tenant_id") || "default";
  return proxyRequest("themis", `/policies/${id}?tenant_id=${encodeURIComponent(tenantId)}`, z.any());
}

export async function PUT(req: NextRequest, { params }: Params) {
  const { id } = await params;
  try {
    const body = await req.json();
    return proxyRequest("themis", `/policies/${id}`, z.any(), {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to update policy";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}

export async function DELETE(req: NextRequest, { params }: Params) {
  const { id } = await params;
  const tenantId = req.nextUrl.searchParams.get("tenant_id") || "default";
  return proxyRequest("themis", `/policies/${id}?tenant_id=${encodeURIComponent(tenantId)}`, z.any(), {
    method: "DELETE",
  });
}
