import { NextRequest, NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";

const NEXUS_LOCAL_FALLBACK = "http://localhost:8080";

export async function GET(req: NextRequest) {
  const nexusUrl = getServiceUrl("nexus");
  const namespace = req.nextUrl.searchParams.get("namespace") || "";
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : "";

  try {
    const res = await fetch(`${nexusUrl}/tuples${qs}`, { cache: "no-store" }).catch(async () => {
      return await fetch(`${NEXUS_LOCAL_FALLBACK}/tuples${qs}`, { cache: "no-store" });
    });
    const data = await res.json();
    return NextResponse.json(data || []);
  } catch (err: any) {
    return NextResponse.json([]);
  }
}

export async function POST(req: NextRequest) {
  const nexusUrl = getServiceUrl("nexus");

  try {
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

    let res = await fetch(`${nexusUrl}/tuples`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    }).catch(async () => {
      return await fetch(`${NEXUS_LOCAL_FALLBACK}/tuples`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
    });

    const data = await res.json();
    if (!res.ok) {
      return NextResponse.json(data, { status: res.status });
    }
    return NextResponse.json(data, { status: 201 });
  } catch (err: any) {
    return NextResponse.json({ error: err.message }, { status: 500 });
  }
}

export async function DELETE(req: NextRequest) {
  const nexusUrl = getServiceUrl("nexus");

  try {
    const body = await req.json();
    const payload = { tuples: body.tuples || [body] };

    let res = await fetch(`${nexusUrl}/tuples`, {
      method: "DELETE",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    }).catch(async () => {
      return await fetch(`${NEXUS_LOCAL_FALLBACK}/tuples`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
    });

    const data = await res.json();
    if (!res.ok) {
      return NextResponse.json(data, { status: res.status });
    }
    return NextResponse.json(data);
  } catch (err: any) {
    return NextResponse.json({ error: err.message }, { status: 500 });
  }
}
