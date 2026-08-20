import { NextResponse } from "next/server";
import { randomUUID } from "crypto";
import { getServiceUrl } from "@/lib/api-config";

// Hermes's /saml/metadata returns raw SAML metadata XML, not JSON, so it
// can't go through proxyRequest (which validates a JSON body against a Zod
// schema). This still propagates the upstream's real status/body instead of
// masking a failure, matching what proxyRequest does for JSON routes.
export async function GET() {
  const requestId = randomUUID();
  const headers = { "x-request-id": requestId };
  const url = `${getServiceUrl("hermes")}/saml/metadata`;

  let res: Response;
  try {
    res = await fetch(url, { cache: "no-store" });
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
