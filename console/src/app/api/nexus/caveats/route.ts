import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";
import { caveatListSchema, caveatSchema } from "@/lib/api/schemas/nexus";

export async function GET() {
  return proxyRequest("nexus", "/admin/caveats", caveatListSchema);
}

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const { name, cel_expression } = body;

    if (!name || !cel_expression) {
      return NextResponse.json(
        { error: "name and cel_expression are required" },
        { status: 400 }
      );
    }

    return proxyRequest("nexus", "/admin/caveats", caveatSchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, cel_expression }),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to create caveat";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}

export async function DELETE(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const name = searchParams.get("name");

  if (!name) {
    return NextResponse.json(
      { error: "caveat name parameter is required" },
      { status: 400 }
    );
  }

  return proxyRequest("nexus", `/admin/caveats/${encodeURIComponent(name)}`, z.object({ status: z.string() }), {
    method: "DELETE",
  });
}
