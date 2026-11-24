import { NextResponse } from "next/server";
import { requireOperator } from "@/lib/auth/session";

export async function GET() {
  const auth = await requireOperator("observability:read");
  if (!auth.ok) return auth.response;
  return NextResponse.json({ error: "SLO recording rules are not configured", source: "prometheus" }, { status: 503 });
}
