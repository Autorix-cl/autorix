import { NextRequest } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";

type Params = { params: Promise<{ id: string }> };

export async function DELETE(_req: NextRequest, { params }: Params) {
  const { id } = await params;
  return proxyRequest("ego", `/admin/sessions/${id}`, z.any(), {
    method: "DELETE",
  });
}
