import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { verifyKeyResponseSchema } from "@/lib/api/schemas/vulcan";

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const { macaroon, context } = body;

    if (!macaroon) {
      return NextResponse.json({ error: "macaroon is required" }, { status: 400 });
    }

    const payload = {
      macaroon,
      context: context || {
        now: new Date().toISOString(),
        ip_address: "127.0.0.1",
        method: "GET",
        path: "/",
      },
    };

    return proxyRequest("vulcan", "/keys/verify", verifyKeyResponseSchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Verification request failed";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
