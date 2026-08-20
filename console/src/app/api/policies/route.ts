import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { policyListSchema, policySchema } from "@/lib/api/schemas/themis";

export async function GET(req: NextRequest) {
  const tenantId = req.nextUrl.searchParams.get("tenant_id") || "default";
  return proxyRequest("themis", `/policies?tenant_id=${encodeURIComponent(tenantId)}`, policyListSchema);
}

export async function POST(req: NextRequest) {
  const body = await req.json();
  const payload = {
    tenant_id: body.tenantId || "default",
    name: body.name,
    description: body.description || "",
    expression: body.expression,
    priority: Number(body.priority) || 1,
    enabled: body.enabled !== false,
    labels: body.labels || {},
  };

  return proxyRequest("themis", "/policies", policySchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}
