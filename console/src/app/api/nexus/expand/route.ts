import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { expandResponseSchema } from "@/lib/api/schemas/nexus";

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const { namespace, object, relation } = body;

    if (!namespace || !object || !relation) {
      return NextResponse.json(
        { error: "namespace, object, and relation are required" },
        { status: 400 }
      );
    }

    return proxyRequest("nexus", "/expand", expandResponseSchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ namespace, object, relation }),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to expand relation";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
