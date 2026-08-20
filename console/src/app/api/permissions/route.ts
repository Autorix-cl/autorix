import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { deleteTuplesResponseSchema, tupleListSchema, writeTuplesResponseSchema } from "@/lib/api/schemas/nexus";

export async function GET(req: NextRequest) {
  const namespace = req.nextUrl.searchParams.get("namespace") || "";
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : "";
  return proxyRequest("nexus", `/tuples${qs}`, tupleListSchema);
}

export async function POST(req: NextRequest) {
  const body = await req.json();
  const payload = {
    tuples: [
      {
        namespace: body.namespace,
        object: body.object,
        relation: body.relation,
        subject_namespace: body.subjectNamespace || "user",
        subject_id: body.subjectId,
        subject_relation: body.subjectRelation || "",
        caveat_name: body.caveatName || "",
        caveat_context: body.caveatContext || undefined,
      },
    ],
  };

  return proxyRequest("nexus", "/tuples", writeTuplesResponseSchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function DELETE(req: NextRequest) {
  const body = await req.json();
  const payload = { tuples: body.tuples || [body] };

  return proxyRequest("nexus", "/tuples", deleteTuplesResponseSchema, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}
