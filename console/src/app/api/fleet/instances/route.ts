import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const queryString = searchParams.toString() ? `?${searchParams.toString()}` : "";
  return proxyRequest("argus", `/v1/instances${queryString}`, z.unknown(), {
    method: "GET",
    requiredPermission: "fleet:read",
  });
}
