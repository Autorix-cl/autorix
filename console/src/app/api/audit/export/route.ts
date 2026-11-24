import { NextRequest, NextResponse } from "next/server";
import { randomUUID } from "crypto";
import { getServiceUrl } from "@/lib/api-config";
import { buildInternalAuthHeaders } from "@/lib/api/internal-headers";
import { requireOperator } from "@/lib/auth/session";

const EXPORT_TIMEOUT_MS = 15_000;

export async function GET(request: NextRequest) {
  const requestId = randomUUID();
  const baseHeaders = { "x-request-id": requestId, "x-correlation-id": requestId };

  const auth = await requireOperator("audit:read", baseHeaders);
  if (!auth.ok) return auth.response;

  const { searchParams } = new URL(request.url);
  const queryString = searchParams.toString() ? `?${searchParams.toString()}` : "";
  const url = `${getServiceUrl("argus")}/v1/audit/export${queryString}`;

  const upstreamHeaders = buildInternalAuthHeaders({
    "x-request-id": requestId,
    "x-correlation-id": requestId,
  });

  let res: Response;
  try {
    res = await fetch(url, {
      method: "GET",
      headers: upstreamHeaders,
      cache: "no-store",
      signal: AbortSignal.timeout(EXPORT_TIMEOUT_MS),
    });
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? `argus is unreachable: ${err.message}` : "argus is unreachable" },
      { status: 502, headers: baseHeaders },
    );
  }

  const rawBody = await res.arrayBuffer();
  const contentType =
    res.headers.get("content-type") ||
    (searchParams.get("format") === "csv" ? "text/csv; charset=utf-8" : "application/json");
  const contentDisposition = res.headers.get("content-disposition");

  const responseHeaders: Record<string, string> = {
    ...baseHeaders,
    "Content-Type": contentType,
  };
  if (contentDisposition) {
    responseHeaders["Content-Disposition"] = contentDisposition;
  }

  return new NextResponse(rawBody, {
    status: res.status,
    headers: responseHeaders,
  });
}
