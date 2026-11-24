import { NextResponse } from "next/server";
import { randomUUID } from "crypto";
import { getServiceUrl } from "@/lib/api-config";
import { requireOperator } from "@/lib/auth/session";

const METADATA_TIMEOUT_MS = 8_000;

// Hermes's /saml/metadata returns raw SAML metadata XML, not JSON, so it
// can't go through proxyRequest (which validates a JSON body against a Zod
// schema). This still enforces authentication/RBAC and propagates the upstream's
// real status/body instead of masking a failure.
export async function GET() {
  const requestId = randomUUID();
  const headers = { "x-request-id": requestId };

  const auth = await requireOperator("enterprise:read", headers);
  if (!auth.ok) return auth.response;

  const url = `${getServiceUrl("hermes")}/saml/metadata`;

  let res: Response;
  try {
    res = await fetch(url, {
      cache: "no-store",
      signal: AbortSignal.timeout(METADATA_TIMEOUT_MS),
    });
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? `hermes is unreachable: ${err.message}` : "hermes is unreachable" },
      { status: 502, headers },
    );
  }

  const body = await res.text();
  const contentType = res.headers.get("content-type") || "application/xml";
  return new NextResponse(body, {
    status: res.status,
    headers: { ...headers, "Content-Type": contentType },
  });
}
