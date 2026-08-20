import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { oauth2ClientListSchema, oauth2ClientSchema } from "@/lib/api/schemas/oauth2";

export async function GET() {
  return proxyRequest("janus", "/admin/clients", oauth2ClientListSchema);
}

export async function POST(req: NextRequest) {
  const body = await req.json();
  const payload = {
    client_id: body.clientId,
    client_name: body.clientName,
    client_secret: body.clientSecret || "",
    is_public: Boolean(body.isPublic),
    scopes: Array.isArray(body.scopes) ? body.scopes : (body.scopes || "").split(" ").filter(Boolean),
    grant_types: body.isPublic ? ["authorization_code", "refresh_token"] : ["client_credentials"],
    redirect_uris: body.redirectUris || [],
    response_types: body.isPublic ? ["code"] : [],
  };

  return proxyRequest("janus", "/admin/clients", oauth2ClientSchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}
