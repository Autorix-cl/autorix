import { NextRequest } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url);
  const queryString = searchParams.toString() ? `?${searchParams.toString()}` : "";
  return proxyRequest("argus", `/admin/audit${queryString}`, z.unknown(), {
    method: "GET",
    requiredPermission: "audit:read",
  });
}

export async function POST(request: NextRequest) {
  const body = await request.json();
  return proxyRequest("argus", "/admin/audit", z.unknown(), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    requiredPermission: "audit:write",
  });
}
