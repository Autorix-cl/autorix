import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { rotateKeyResponseSchema } from "@/lib/api/schemas/vulcan";

type Params = { params: Promise<{ id: string }> };

export async function POST(req: NextRequest, { params }: Params) {
  const { id } = await params;
  try {
    let gracePeriod = "24h";
    try {
      const body = await req.json();
      if (body.grace_period) gracePeriod = body.grace_period;
    } catch {
      // payload is optional
    }

    return proxyRequest("vulcan", `/admin/keys/${id}/rotate`, rotateKeyResponseSchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ grace_period: gracePeriod }),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Key rotation failed";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
