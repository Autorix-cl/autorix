import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { identitySchemaListSchema, identitySchemaDefinitionSchema } from "@/lib/api/schemas/identity";

export async function GET() {
  return proxyRequest("ego", "/admin/schemas", identitySchemaListSchema);
}

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    return proxyRequest("ego", "/admin/schemas", identitySchemaDefinitionSchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to create schema";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
