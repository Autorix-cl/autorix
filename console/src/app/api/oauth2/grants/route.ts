import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { oauth2GrantListSchema } from "@/lib/api/schemas/oauth2";

export async function GET(req: NextRequest) {
  const search = req.nextUrl.search;
  return proxyRequest("janus", `/admin/grants${search}`, oauth2GrantListSchema);
}
