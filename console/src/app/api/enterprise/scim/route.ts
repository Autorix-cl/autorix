import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { scimListResponseSchema, scimUserSchema } from "@/lib/api/schemas/hermes";

export async function GET() {
  return proxyRequest("hermes", "/scim/v2/Users", scimListResponseSchema);
}

export async function POST(req: NextRequest) {
  const body = await req.json();
  return proxyRequest("hermes", "/scim/v2/Users", scimUserSchema, {
    method: "POST",
    headers: { "Content-Type": "application/scim+json" },
    body: JSON.stringify(body),
  });
}
