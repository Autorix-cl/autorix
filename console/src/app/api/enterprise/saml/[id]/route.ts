import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";
import { samlProviderSchema } from "@/lib/api/schemas/hermes";

type Params = { params: Promise<{ id: string }> };

export async function GET(req: NextRequest, { params }: Params) {
  const { id } = await params;
  return proxyRequest("hermes", `/admin/providers/${encodeURIComponent(id)}`, samlProviderSchema);
}

export async function PATCH(req: NextRequest, { params }: Params) {
  const { id } = await params;
  try {
    const body = await req.json();
    return proxyRequest("hermes", `/admin/providers/${encodeURIComponent(id)}`, samlProviderSchema, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to update provider";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}

export async function DELETE(req: NextRequest, { params }: Params) {
  const { id } = await params;
  return proxyRequest("hermes", `/admin/providers/${encodeURIComponent(id)}`, z.any(), {
    method: "DELETE",
  });
}
