import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";
import { identitySchemaDefinitionSchema } from "@/lib/api/schemas/identity";

type Params = { params: Promise<{ id: string }> };

export async function GET(_req: NextRequest, { params }: Params) {
  const { id } = await params;
  return proxyRequest("ego", `/admin/schemas/${id}`, identitySchemaDefinitionSchema);
}

export async function PATCH(req: NextRequest, { params }: Params) {
  const { id } = await params;
  try {
    const body = await req.json();
    return proxyRequest("ego", `/admin/schemas/${id}`, identitySchemaDefinitionSchema, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to update schema";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}

export async function DELETE(_req: NextRequest, { params }: Params) {
  const { id } = await params;
  return proxyRequest("ego", `/admin/schemas/${id}`, z.any(), {
    method: "DELETE",
  });
}
