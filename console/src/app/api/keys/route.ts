import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { paginatedApiKeyListSchema, createKeyResponseSchema } from "@/lib/api/schemas/vulcan";

export async function GET(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const cursor = searchParams.get("cursor") || "";
  const limit = searchParams.get("limit") || "50";

  const query = new URLSearchParams();
  if (cursor) query.set("cursor", cursor);
  if (limit) query.set("limit", limit);

  const qs = query.toString() ? `?${query.toString()}` : "";
  return proxyRequest("vulcan", `/keys${qs}`, paginatedApiKeyListSchema);
}

export async function POST(req: NextRequest) {
  const body = await req.json();
  const payload = {
    name: body.name,
    owner_id: body.ownerId || "system",
    is_live: Boolean(body.isLive),
    scopes: body.scopes || [],
  };

  return proxyRequest("vulcan", "/keys", createKeyResponseSchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}
