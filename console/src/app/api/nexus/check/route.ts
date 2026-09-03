import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { checkResponseSchema } from "@/lib/api/schemas/nexus";

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();

    let namespace = body.namespace || "";
    let object = body.object || "";
    const relation = body.relation || "";
    let subjectNamespace = body.subject_namespace || "";
    let subjectId = body.subject_id || "";
    const subjectRelation = body.subject_relation || "";

    // Parse shorthand formatted string (e.g. subject: "user:alice", object: "document:doc_1")
    if (body.subject && !subjectId) {
      const parts = String(body.subject).split(":");
      if (parts.length >= 2) {
        subjectNamespace = parts[0];
        subjectId = parts.slice(1).join(":");
      } else {
        subjectNamespace = "user";
        subjectId = body.subject;
      }
    }

    if (body.object && (!namespace || namespace === object)) {
      const parts = String(body.object).split(":");
      if (parts.length >= 2) {
        namespace = parts[0];
        object = parts.slice(1).join(":");
      }
    }

    if (!namespace || !object || !relation) {
      return NextResponse.json(
        { error: "namespace, object, and relation are required" },
        { status: 400 }
      );
    }

    return proxyRequest("nexus", "/check", checkResponseSchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        namespace,
        object,
        relation,
        subject_namespace: subjectNamespace || "user",
        subject_id: subjectId,
        subject_relation: subjectRelation,
        request_context: body.request_context || {},
        explain: body.explain ?? true,
      }),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to parse check request";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}

