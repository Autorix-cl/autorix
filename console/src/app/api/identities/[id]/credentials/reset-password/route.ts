import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { resetPasswordResultSchema } from "@/lib/api/schemas/identity";

type Params = { params: Promise<{ id: string }> };

export async function POST(req: NextRequest, { params }: Params) {
  const { id } = await params;
  try {
    const body = await req.json().catch(() => ({}));
    return proxyRequest("ego", `/admin/identities/${id}/credentials/reset-password`, resetPasswordResultSchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        password: body.password || undefined,
        force_rotation: Boolean(body.force_rotation ?? true),
      }),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to reset password";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
