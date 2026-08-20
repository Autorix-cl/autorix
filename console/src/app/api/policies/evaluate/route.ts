import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { evaluateResponseSchema } from "@/lib/api/schemas/themis";

export async function POST(req: NextRequest) {
  const body = await req.json();
  const payload = {
    tenant_id: body.tenantId || "default",
    policy_id: body.policyId || "",
    payload: body.payload || {},
    label_filter: body.labelFilter || {},
  };

  return proxyRequest("themis", "/policies/evaluate", evaluateResponseSchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}
