import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";

type Params = { params: Promise<{ id: string }> };

export async function PATCH(req: NextRequest, { params }: Params) {
  const { id } = await params;
  try {
    const body = await req.json();
    const enabled = Boolean(body.enabled);

    // Fetch existing policy first
    const getRes = await proxyRequest("themis", `/policies/${id}`, z.any());
    if (!getRes.ok) {
      return getRes;
    }

    const policy = await getRes.json();
    policy.enabled = enabled;

    return proxyRequest("themis", `/policies/${id}`, z.any(), {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(policy),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to toggle policy";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
