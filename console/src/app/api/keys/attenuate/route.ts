import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { attenuateResponseSchema } from "@/lib/api/schemas/vulcan";

export async function POST(req: NextRequest) {
  const body = await req.json();
  return proxyRequest("vulcan", "/keys/attenuate", attenuateResponseSchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}
