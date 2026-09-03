import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { validationResultSchema } from "@/lib/api/schemas/themis";

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const { expression } = body;

    if (!expression) {
      return NextResponse.json({ error: "expression is required" }, { status: 400 });
    }

    return proxyRequest("themis", "/policies/validate", validationResultSchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ expression }),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Validation failed";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
