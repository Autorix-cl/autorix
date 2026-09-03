import { NextRequest } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";
import { sessionListSchema } from "@/lib/api/schemas/identity";

type Params = { params: Promise<{ id: string }> };

export async function GET(_req: NextRequest, { params }: Params) {
  const { id } = await params;
  return proxyRequest("ego", `/admin/identities/${id}/sessions`, sessionListSchema);
}

export async function DELETE(_req: NextRequest, { params }: Params) {
  const { id } = await params;
  return proxyRequest("ego", `/admin/identities/${id}/sessions`, z.any(), {
    method: "DELETE",
  });
}
