import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { testSuiteResultSchema } from "@/lib/api/schemas/themis";

type Params = { params: Promise<{ id: string }> };

export async function POST(req: NextRequest, { params }: Params) {
  const { id } = await params;
  const tenantId = req.nextUrl.searchParams.get("tenant_id") || "default";

  return proxyRequest(
    "themis",
    `/policies/${id}/test-suite?tenant_id=${encodeURIComponent(tenantId)}`,
    testSuiteResultSchema,
    { method: "POST" }
  );
}
