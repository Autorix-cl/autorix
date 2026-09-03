import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import {
  lookupSubjectsResponseSchema,
  lookupResourcesResponseSchema,
} from "@/lib/api/schemas/nexus";

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const { type, namespace, object, relation, subject_namespace, subject_id, subject_relation } = body;

    if (type === "subjects") {
      if (!namespace || !object || !relation) {
        return NextResponse.json(
          { error: "namespace, object, and relation are required for lookup subjects" },
          { status: 400 }
        );
      }
      return proxyRequest("nexus", "/lookup/subjects", lookupSubjectsResponseSchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ namespace, object, relation }),
      });
    }

    if (type === "resources") {
      if (!namespace || !relation || !subject_id) {
        return NextResponse.json(
          { error: "namespace, relation, and subject_id are required for lookup resources" },
          { status: 400 }
        );
      }
      return proxyRequest("nexus", "/lookup/resources", lookupResourcesResponseSchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          namespace,
          relation,
          subject_namespace: subject_namespace || "user",
          subject_id,
          subject_relation: subject_relation || "",
        }),
      });
    }

    return NextResponse.json(
      { error: "type must be 'subjects' or 'resources'" },
      { status: 400 }
    );
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to execute lookup";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
