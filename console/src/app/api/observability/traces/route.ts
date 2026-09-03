import { NextResponse } from "next/server";
import { telemetryStore } from "@/lib/server/telemetry-store";

export async function GET() {
  const traces = telemetryStore.getTraces(50);
  return NextResponse.json(traces);
}
