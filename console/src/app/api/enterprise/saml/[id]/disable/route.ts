import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { samlProviderSchema } from "@/lib/api/schemas/hermes";

type Params = { params: Promise<{ id: string }> };

export async function POST(req: NextRequest, { params }: Params) {
  const { id } = await params;
  return proxyRequest("hermes", `/admin/providers/${encodeURIComponent(id)}/disable`, samlProviderSchema, {
    method: "POST",
  });
}
