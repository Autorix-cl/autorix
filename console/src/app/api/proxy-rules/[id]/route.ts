import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { deleteRuleResponseSchema, ruleSchema } from "@/lib/api/schemas/aegis";

export async function PUT(req: NextRequest, context: { params: Promise<{ id: string }> }) {
  const { id } = await context.params;
  const body = await req.json();

  return proxyRequest("aegis", `/rules/${id}`, ruleSchema, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

export async function DELETE(req: NextRequest, context: { params: Promise<{ id: string }> }) {
  const { id } = await context.params;

  return proxyRequest("aegis", `/rules/${id}`, deleteRuleResponseSchema, { method: "DELETE" });
}
