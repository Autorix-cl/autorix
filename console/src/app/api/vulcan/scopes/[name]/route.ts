import { NextRequest } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";

type Params = { params: Promise<{ name: string }> };

export async function DELETE(req: NextRequest, { params }: Params) {
  const { name } = await params;
  return proxyRequest("vulcan", `/admin/scopes/${encodeURIComponent(name)}`, z.any(), {
    method: "DELETE",
  });
}
