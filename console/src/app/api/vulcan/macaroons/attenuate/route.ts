import { NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";
import { attenuateResponseSchema } from "@/lib/api/schemas/vulcan";

const attenuateBodySchema = z.object({
  macaroon: z.record(z.string(), z.unknown()),
  caveat: z.string().min(1).optional(),
  caveats: z.array(z.string().min(1)).optional(),
});

// POST /api/vulcan/macaroons/attenuate
// Proxies macaroon attenuation to the Vulcan engine (POST /keys/attenuate).
export async function POST(req: Request) {
  let rawBody: unknown;
  try {
    rawBody = await req.json();
  } catch {
    return NextResponse.json({ error: "Invalid JSON payload" }, { status: 400 });
  }

  const parsed = attenuateBodySchema.safeParse(rawBody);
  if (!parsed.success) {
    return NextResponse.json({ error: "macaroon and at least one caveat are required" }, { status: 400 });
  }

  const caveatsToApply = parsed.data.caveat
    ? [parsed.data.caveat]
    : (parsed.data.caveats ?? []).map((c) => c.trim()).filter(Boolean);

  if (caveatsToApply.length === 0) {
    return NextResponse.json({ error: "At least one caveat predicate is required" }, { status: 400 });
  }

  let currentMacaroon: unknown = parsed.data.macaroon;
  for (const caveat of caveatsToApply) {
    const res = await proxyRequest("vulcan", "/keys/attenuate", attenuateResponseSchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ macaroon: currentMacaroon, caveat }),
    });

    if (!res.ok) {
      return res;
    }

    currentMacaroon = await res.json();
  }

  const token = Buffer.from(JSON.stringify(currentMacaroon)).toString("base64url");
  return NextResponse.json({ macaroon: currentMacaroon, token }, { status: 200 });
}
