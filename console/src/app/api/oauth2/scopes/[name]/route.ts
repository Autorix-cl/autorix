import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { z } from "zod";

interface RouteParams {
  params: Promise<{ name: string }>;
}

export async function DELETE(_req: NextRequest, { params }: RouteParams) {
  const { name } = await params;
  return proxyRequest("janus", `/admin/scopes/${name}`, z.unknown(), {
    method: "DELETE",
  });
}

