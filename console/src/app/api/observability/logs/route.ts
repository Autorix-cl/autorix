import { NextRequest, NextResponse } from "next/server";
import { telemetryStore } from "@/lib/server/telemetry-store";

export async function GET(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const engine = searchParams.get("engine");
  const level = searchParams.get("level");
  const requestId = searchParams.get("request_id");
  const query = searchParams.get("query")?.toLowerCase();
  const limit = Number(searchParams.get("limit")) || 100;

  let filtered = telemetryStore.getLogs({
    engine: engine || undefined,
    level: level || undefined,
    limit,
  });

  if (requestId) {
    filtered = filtered.filter((l) => l.request_id === requestId || l.correlation_id === requestId);
  }
  if (query) {
    filtered = filtered.filter(
      (l) =>
        l.message.toLowerCase().includes(query) ||
        l.engine.toLowerCase().includes(query) ||
        l.request_id?.toLowerCase().includes(query)
    );
  }

  return NextResponse.json(filtered);
}
