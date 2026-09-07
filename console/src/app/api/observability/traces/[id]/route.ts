import { NextResponse } from "next/server";
export async function GET() {
  return NextResponse.json({ error: "A trace backend is not configured", source: "traces" }, { status: 501 });
}
