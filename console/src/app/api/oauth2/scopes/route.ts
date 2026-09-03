import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { oauth2ScopeListSchema, oauth2ScopeSchema } from "@/lib/api/schemas/oauth2";

export async function GET() {
  return proxyRequest("janus", "/admin/scopes", oauth2ScopeListSchema);
}

export async function POST(req: NextRequest) {
  const body = await req.text();
  return proxyRequest("janus", "/admin/scopes", oauth2ScopeSchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body,
  });
}
