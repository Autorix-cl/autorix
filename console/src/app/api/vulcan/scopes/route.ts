import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { scopeListSchema, scopeSchema } from "@/lib/api/schemas/vulcan";

export async function GET() {
  return proxyRequest("vulcan", "/admin/scopes", scopeListSchema);
}

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const { name, description } = body;

    if (!name) {
      return NextResponse.json({ error: "scope name is required" }, { status: 400 });
    }

    return proxyRequest("vulcan", "/admin/scopes", scopeSchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, description: description || "" }),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to create scope";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
