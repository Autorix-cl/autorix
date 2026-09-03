import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { recoveryLinkResultSchema } from "@/lib/api/schemas/identity";

type Params = { params: Promise<{ id: string }> };

export async function POST(req: NextRequest, { params }: Params) {
  const { id } = await params;
  try {
    const body = await req.json().catch(() => ({}));
    return proxyRequest("ego", `/admin/identities/${id}/recovery-link`, recoveryLinkResultSchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        expires_in: body.expires_in || "1h",
      }),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to issue recovery link";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
