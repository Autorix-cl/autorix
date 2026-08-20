import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { deletePolicyResponseSchema, policySchema } from "@/lib/api/schemas/themis";

export async function DELETE(req: NextRequest, context: { params: Promise<{ id: string }> }) {
  const { id } = await context.params;
  const tenantId = req.nextUrl.searchParams.get("tenant_id") || "default";

  return proxyRequest(
    "themis",
    `/policies/${id}?tenant_id=${encodeURIComponent(tenantId)}`,
    deletePolicyResponseSchema,
    { method: "DELETE" },
  );
}

export async function PUT(req: NextRequest, context: { params: Promise<{ id: string }> }) {
  const { id } = await context.params;
  const body = await req.json();

  return proxyRequest("themis", `/policies/${id}`, policySchema, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}
