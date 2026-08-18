import { NextRequest, NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";

const NEXUS_LOCAL_FALLBACK = "http://localhost:8080";

export async function POST(req: NextRequest) {
  const nexusUrl = getServiceUrl("nexus");

  try {
    const body = await req.json();
    const payload = {
      namespace: body.namespace,
      object: body.object,
      relation: body.relation,
      subject_namespace: "user",
      subject_id: body.subjectId,
      request_context: body.requestContext || undefined,
    };

    let res = await fetch(`${nexusUrl}/check`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    }).catch(async () => {
      return await fetch(`${NEXUS_LOCAL_FALLBACK}/check`, {
        method: "POST",
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
