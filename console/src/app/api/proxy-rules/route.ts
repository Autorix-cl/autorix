import { NextRequest, NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";

const AEGIS_LOCAL_FALLBACK = "http://localhost:4456";

export async function GET() {
  const aegisUrl = getServiceUrl("aegis");

  try {
    const res = await fetch(`${aegisUrl}/rules`, { cache: "no-store" }).catch(async () => {
      return await fetch(`${AEGIS_LOCAL_FALLBACK}/rules`, { cache: "no-store" });
    });
    const data = await res.json();
    return NextResponse.json(data || []);
  } catch (err: any) {
    return NextResponse.json([]);
  }
}

export async function POST(req: NextRequest) {
  const aegisUrl = getServiceUrl("aegis");

  try {
    const body = await req.json();

    let res = await fetch(`${aegisUrl}/rules`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }).catch(async () => {
      return await fetch(`${AEGIS_LOCAL_FALLBACK}/rules`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
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
