import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { policyFixtureListSchema, policyFixtureSchema } from "@/lib/api/schemas/themis";

type Params = { params: Promise<{ id: string }> };

export async function GET(req: NextRequest, { params }: Params) {
  const { id } = await params;
  const tenantId = req.nextUrl.searchParams.get("tenant_id") || "default";
  return proxyRequest(
    "themis",
    `/policies/${id}/fixtures?tenant_id=${encodeURIComponent(tenantId)}`,
    policyFixtureListSchema
  );
}

export async function POST(req: NextRequest, { params }: Params) {
  const { id } = await params;
  try {
    const body = await req.json();
    const tenantId = req.nextUrl.searchParams.get("tenant_id") || body.tenant_id || "default";

    const payload = {
      tenant_id: tenantId,
      name: body.name,
      description: body.description || "",
      payload: body.payload || {},
      expected_result: Boolean(body.expected_result),
    };

    return proxyRequest("themis", `/policies/${id}/fixtures?tenant_id=${encodeURIComponent(tenantId)}`, policyFixtureSchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to create fixture";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
