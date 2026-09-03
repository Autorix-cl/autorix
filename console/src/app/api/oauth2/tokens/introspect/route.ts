import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { introspectResponseSchema } from "@/lib/api/schemas/oauth2";

export async function POST(req: NextRequest) {
  let token = "";
  let tokenTypeHint = "";

  const contentType = req.headers.get("content-type") || "";
  if (contentType.includes("application/json")) {
    const json = await req.json();
    token = json.token || "";
    tokenTypeHint = json.token_type_hint || "";
  } else {
    const formData = await req.formData();
    token = (formData.get("token") as string) || "";
    tokenTypeHint = (formData.get("token_type_hint") as string) || "";
  }

  const params = new URLSearchParams();
  params.set("token", token);
  if (tokenTypeHint) {
    params.set("token_type_hint", tokenTypeHint);
  }

  return proxyRequest("janus", "/oauth2/introspect", introspectResponseSchema, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: params.toString(),
  });
}
