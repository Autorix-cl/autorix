import { NextRequest } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";
import { mfaStatusSchema } from "@/lib/api/schemas/identity";

type Params = { params: Promise<{ id: string }> };

export async function GET(_req: NextRequest, { params }: Params) {
  const { id } = await params;
  return proxyRequest("ego", `/admin/identities/${id}/mfa`, mfaStatusSchema);
}

export async function DELETE(_req: NextRequest, { params }: Params) {
  const { id } = await params;
  return proxyRequest("ego", `/admin/identities/${id}/mfa`, z.any(), {
    method: "DELETE",
  });
}
