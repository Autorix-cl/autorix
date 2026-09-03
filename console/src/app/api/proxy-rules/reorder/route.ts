import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { reorderRulesRequestSchema, reorderResponseSchema } from "@/lib/api/schemas/aegis";

export async function PUT(req: NextRequest) {
  let body: unknown;
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "invalid json body" }, { status: 400 });
  }

  const parsed = reorderRulesRequestSchema.safeParse(body);
  if (!parsed.success) {
    return NextResponse.json(
      { error: "invalid reorder request: ids array required", details: parsed.error.issues },
      { status: 400 }
    );
  }

  return proxyRequest("aegis", "/rules/reorder", reorderResponseSchema, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(parsed.data),
  });
}
