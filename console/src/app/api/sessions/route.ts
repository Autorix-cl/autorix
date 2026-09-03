import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { paginatedSessionListSchema } from "@/lib/api/schemas/identity";

export async function GET(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const cursor = searchParams.get("cursor") || "";
  const limit = searchParams.get("limit") || "50";

  const qs = new URLSearchParams();
  if (cursor) qs.set("cursor", cursor);
  if (limit) qs.set("limit", limit);

  return proxyRequest("ego", `/admin/sessions?${qs.toString()}`, paginatedSessionListSchema);
}
