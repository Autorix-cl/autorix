import { NextResponse } from "next/server";
export async function GET() {
  return NextResponse.json({ error: "A log aggregation backend is not configured", source: "logs" }, { status: 501 });
}
