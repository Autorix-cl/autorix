import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";

export async function GET() {
  return proxyRequest("argus", "/v1/enrollment-tokens", z.unknown(), {
    method: "GET",
    requiredPermission: "fleet:read",
  });
}

export async function POST(request: Request) {
  const body = await request.json();
  return proxyRequest("argus", "/v1/enrollment-tokens", z.unknown(), {
    method: "POST",
    body: JSON.stringify(body),
    requiredPermission: "fleet:admin",
  });
}
