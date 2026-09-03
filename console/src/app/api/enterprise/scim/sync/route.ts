import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { scimSyncHistoryListSchema, scimSyncHistorySchema } from "@/lib/api/schemas/hermes";

export async function GET() {
  return proxyRequest("hermes", "/admin/scim/sync-history", scimSyncHistoryListSchema);
}

export async function POST(req: NextRequest) {
  try {
    let payload = {
      resource_type: "Users",
      status: "success",
      total_records: 0,
      created_count: 0,
      updated_count: 0,
      deleted_count: 0,
      error_count: 0,
      errors: [] as string[],
    };

    try {
      const body = await req.json();
      payload = { ...payload, ...body };
    } catch {
      // payload is optional
    }

    return proxyRequest("hermes", "/admin/scim/sync-history", scimSyncHistorySchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to record sync";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
