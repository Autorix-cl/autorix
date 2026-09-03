import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { credentialListSchema } from "@/lib/api/schemas/identity";

type Params = { params: Promise<{ id: string }> };

export async function GET(_req: NextRequest, { params }: Params) {
  const { id } = await params;
  return proxyRequest("ego", `/admin/identities/${id}/credentials`, credentialListSchema);
}
