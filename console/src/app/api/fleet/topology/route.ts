import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";

export async function GET() {
  return proxyRequest("argus", "/v1/topology", z.unknown(), {
    method: "GET",
    requiredPermission: "fleet:read",
  });
}
