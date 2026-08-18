import { NextRequest, NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";

const AEGIS_LOCAL_FALLBACK = "http://localhost:4456";

export async function POST(req: NextRequest) {
  const aegisUrl = getServiceUrl("aegis");

  try {
    const body = await req.json();
    const payload = { method: body.method, path: body.path };

    let res = await fetch(`${aegisUrl}/rules/test-match`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    }).catch(async () => {
      return await fetch(`${AEGIS_LOCAL_FALLBACK}/rules/test-match`, {
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
