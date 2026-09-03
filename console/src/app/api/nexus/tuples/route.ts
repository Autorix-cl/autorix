import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import {
  tupleListSchema,
  writeTuplesResponseSchema,
  deleteTuplesResponseSchema,
} from "@/lib/api/schemas/nexus";

export async function GET(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const namespace = searchParams.get("namespace") || "";
  const limit = searchParams.get("limit") || "100";
  const cursor = searchParams.get("cursor") || "";

  const qs = new URLSearchParams();
  if (namespace) qs.set("namespace", namespace);
  if (limit) qs.set("limit", limit);
  if (cursor) qs.set("cursor", cursor);

  const path = `/tuples${qs.toString() ? `?${qs.toString()}` : ""}`;
  return proxyRequest("nexus", path, tupleListSchema);
}

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    let tuples = Array.isArray(body) ? body : body.tuples ? body.tuples : [body];

    // Normalize property names if incoming from camelCase components
    tuples = tuples.map((t: Record<string, unknown>) => ({
      namespace: t.namespace || t.objectType || "",
      object: t.object || t.objectId || "",
      relation: t.relation || "",
      subject_namespace: t.subject_namespace || t.subjectType || "user",
      subject_id: t.subject_id || t.subjectId || "",
      subject_relation: t.subject_relation || t.subjectRelation || "",
      caveat_name: t.caveat_name || t.caveatName || undefined,
      caveat_context: t.caveat_context || t.caveatContext || undefined,
    }));

    return proxyRequest("nexus", "/tuples", writeTuplesResponseSchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ tuples }),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to parse write tuples request";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}

export async function DELETE(req: NextRequest) {
  try {
    const body = await req.json().catch(() => ({}));
    let tuples = Array.isArray(body) ? body : body.tuples ? body.tuples : [];

    // Also support searchParams ?ids= or query-based deletes
    if (tuples.length === 0) {
      const { searchParams } = new URL(req.url);
      const idsParam = searchParams.get("ids");
      if (idsParam) {
        const idList = idsParam.split(",");
        tuples = idList.map((idStr) => {
          // Parse format: namespace:object#relation@subject_ns:subject_id
          const [resPart, subPart] = idStr.split("@");
          const [nsObj, rel] = (resPart || "").split("#");
          const [ns, obj] = (nsObj || "").split(":");
          const [subNs, subId] = (subPart || "").split(":");
          return {
            namespace: ns || "",
            object: obj || "",
            relation: rel || "",
            subject_namespace: subNs || "user",
            subject_id: subId || "",
          };
        });
      }
    }

    tuples = tuples.map((t: Record<string, unknown>) => ({
      namespace: t.namespace || t.objectType || "",
      object: t.object || t.objectId || "",
      relation: t.relation || "",
      subject_namespace: t.subject_namespace || t.subjectType || "user",
      subject_id: t.subject_id || t.subjectId || "",
      subject_relation: t.subject_relation || t.subjectRelation || "",
    }));

    return proxyRequest("nexus", "/tuples", deleteTuplesResponseSchema, {
      method: "DELETE",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ tuples }),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to delete tuples";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}

