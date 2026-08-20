import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { checkResponseSchema } from "@/lib/api/schemas/nexus";

export async function POST(req: NextRequest) {
  const body = await req.json();
  const payload = {
    namespace: body.namespace,
    object: body.object,
    relation: body.relation,
    subject_namespace: "user",
    subject_id: body.subjectId,
    request_context: body.requestContext || undefined,
  };

  return proxyRequest("nexus", "/check", checkResponseSchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}
