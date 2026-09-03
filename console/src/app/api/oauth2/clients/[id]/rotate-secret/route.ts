import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { rotateSecretResponseSchema } from "@/lib/api/schemas/oauth2";

interface RouteParams {
  params: Promise<{ id: string }>;
}

export async function POST(req: NextRequest, { params }: RouteParams) {
  const { id } = await params;
  const body = await req.text();
  return proxyRequest(
    "janus",
    `/admin/clients/${encodeURIComponent(id)}/rotate-secret`,
    rotateSecretResponseSchema,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: body || "{}",
    }
  );
}
