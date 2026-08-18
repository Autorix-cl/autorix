import { NextRequest, NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";

const AEGIS_LOCAL_FALLBACK = "http://localhost:4456";

export async function PUT(
  req: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  const { id } = await context.params;
  const aegisUrl = getServiceUrl("aegis");

  try {
    const body = await req.json();

    let res = await fetch(`${aegisUrl}/rules/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }).catch(async () => {
      return await fetch(`${AEGIS_LOCAL_FALLBACK}/rules/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
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

export async function DELETE(
  req: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  const { id } = await context.params;
  const aegisUrl = getServiceUrl("aegis");

  try {
    let res = await fetch(`${aegisUrl}/rules/${id}`, { method: "DELETE" }).catch(async () => {
      return await fetch(`${AEGIS_LOCAL_FALLBACK}/rules/${id}`, { method: "DELETE" });
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
