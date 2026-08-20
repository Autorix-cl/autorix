import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { ruleListSchema, ruleSchema } from "@/lib/api/schemas/aegis";

export async function GET() {
  return proxyRequest("aegis", "/rules", ruleListSchema);
}

export async function POST(req: NextRequest) {
  const body = await req.json();

  return proxyRequest("aegis", "/rules", ruleSchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}
