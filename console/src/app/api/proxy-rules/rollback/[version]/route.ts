import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { rollbackResponseSchema } from "@/lib/api/schemas/aegis";

export async function POST(_req: NextRequest, context: { params: Promise<{ version: string }> }) {
  const { version } = await context.params;

  return proxyRequest("aegis", `/rules/rollback/${version}`, rollbackResponseSchema, {
    method: "POST",
  });
}
