import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { oauth2ClientSchema } from "@/lib/api/schemas/oauth2";
import { z } from "zod";

interface RouteParams {
  params: Promise<{ id: string }>;
}

export async function GET(_req: NextRequest, { params }: RouteParams) {
  const { id } = await params;
  return proxyRequest("janus", `/admin/clients/${encodeURIComponent(id)}`, oauth2ClientSchema);
}

export async function PATCH(req: NextRequest, { params }: RouteParams) {
  const { id } = await params;
  const input = await req.json();
  const allowedAudiences = input.allowedAudiences ?? input.allowed_audiences;
  const body = JSON.stringify({
    ...input,
    ...(allowedAudiences !== undefined
      ? {
          allowed_audiences: Array.isArray(allowedAudiences)
            ? allowedAudiences
            : String(allowedAudiences).split(/\s+/).filter(Boolean),
        }
      : {}),
  });
  return proxyRequest("janus", `/admin/clients/${encodeURIComponent(id)}`, oauth2ClientSchema, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body,
  });
}

export async function DELETE(_req: NextRequest, { params }: RouteParams) {
  const { id } = await params;
  return proxyRequest(
    "janus",
    `/admin/clients/${encodeURIComponent(id)}`,
    z.unknown(),
    {
      method: "DELETE",
    }
  );
}
