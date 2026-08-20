import { NextRequest } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url);
  const queryString = searchParams.toString() ? `?${searchParams.toString()}` : "";
  return proxyRequest("argus", `/admin/audit/export${queryString}`, z.unknown(), {
    method: "GET",
    requiredPermission: "audit:read",
  });
}
