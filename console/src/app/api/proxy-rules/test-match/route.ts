import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { testMatchResponseSchema } from "@/lib/api/schemas/aegis";

export async function POST(req: NextRequest) {
  const body = await req.json();
  const payload = { method: body.method, path: body.path };

  return proxyRequest("aegis", "/rules/test-match", testMatchResponseSchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}
