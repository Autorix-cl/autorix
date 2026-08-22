import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { paginatedPolicyListSchema, policySchema } from "@/lib/api/schemas/themis";

export async function GET(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const tenantId = searchParams.get("tenant_id") || "default";
  const cursor = searchParams.get("cursor") || "";
  const limit = searchParams.get("limit") || "50";

  const query = new URLSearchParams();
  query.set("tenant_id", tenantId);
  if (cursor) query.set("cursor", cursor);
  if (limit) query.set("limit", limit);

  return proxyRequest("themis", `/policies?${query.toString()}`, paginatedPolicyListSchema);
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
