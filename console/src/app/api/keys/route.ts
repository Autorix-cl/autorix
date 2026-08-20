import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { apiKeyListSchema, createKeyResponseSchema } from "@/lib/api/schemas/vulcan";

export async function GET() {
  return proxyRequest("vulcan", "/keys", apiKeyListSchema);
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
