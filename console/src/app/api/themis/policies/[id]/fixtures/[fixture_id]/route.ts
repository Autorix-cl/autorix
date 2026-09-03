import { NextRequest } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";

type Params = { params: Promise<{ id: string; fixture_id: string }> };

export async function DELETE(req: NextRequest, { params }: Params) {
  const { id, fixture_id } = await params;
  const tenantId = req.nextUrl.searchParams.get("tenant_id") || "default";

  return proxyRequest(
    "themis",
    `/policies/${id}/fixtures/${fixture_id}?tenant_id=${encodeURIComponent(tenantId)}`,
    z.any(),
    { method: "DELETE" }
  );
}
