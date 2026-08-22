import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { deleteTuplesResponseSchema, paginatedTupleListSchema, writeTuplesResponseSchema } from "@/lib/api/schemas/nexus";

export async function GET(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const namespace = searchParams.get("namespace") || "";
  const cursor = searchParams.get("cursor") || "";
  const limit = searchParams.get("limit") || "50";

  const query = new URLSearchParams();
  if (namespace) query.set("namespace", namespace);
  if (cursor) query.set("cursor", cursor);
  if (limit) query.set("limit", limit);

  const qs = query.toString() ? `?${query.toString()}` : "";
  return proxyRequest("nexus", `/tuples${qs}`, paginatedTupleListSchema);
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
