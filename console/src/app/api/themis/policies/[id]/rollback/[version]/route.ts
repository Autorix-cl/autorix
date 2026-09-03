import { NextRequest } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";

type Params = { params: Promise<{ id: string; version: string }> };

export async function POST(req: NextRequest, { params }: Params) {
  const { id, version } = await params;
  const tenantId = req.nextUrl.searchParams.get("tenant_id") || "default";

  return proxyRequest(
    "themis",
    `/policies/${id}/rollback/${version}?tenant_id=${encodeURIComponent(tenantId)}`,
    z.any(),
    { method: "POST" }
  );
}
