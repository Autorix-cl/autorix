import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";
import { operatorSchema } from "@/lib/api/schemas/operator";

type Params = { params: Promise<{ id: string }> };

export async function PATCH(req: NextRequest, { params }: Params) {
  const { id } = await params;
  try {
    const body = await req.json();
    return proxyRequest("argus", `/v1/operators/${id}`, operatorSchema, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      requiredPermission: "identities:write",
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Invalid update payload";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}

export async function DELETE(_req: NextRequest, { params }: Params) {
  const { id } = await params;
  return proxyRequest("argus", `/v1/operators/${id}`, z.any(), {
    method: "DELETE",
    requiredPermission: "identities:write",
  });
}
