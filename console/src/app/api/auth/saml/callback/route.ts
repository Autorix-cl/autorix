import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { timingSafeEqual } from "crypto";
import { getServiceUrl } from "@/lib/api-config";
import { SESSION_COOKIE_NAME } from "@/lib/auth/session";

const SAML_VERIFY_TIMEOUT_MS = 10_000;

function safeEqual(a: string, b: string): boolean {
  const bufA = Buffer.from(a, "utf8");
  const bufB = Buffer.from(b, "utf8");
  if (bufA.length !== bufB.length) return false;
  return timingSafeEqual(bufA, bufB);
}

function failRedirect(requestUrl: string): NextResponse {
  const response = NextResponse.redirect(new URL("/login?error=saml_failed", requestUrl));
  response.cookies.delete("saml_relay_state");
  return response;
}

export async function POST(request: Request) {
  try {
    const formData = await request.formData();
    const samlResponse = formData.get("SAMLResponse");
    const relayState = formData.get("RelayState");

    const cookieStore = await cookies();
    const savedRelay = cookieStore.get("saml_relay_state")?.value;

    if (
      typeof samlResponse !== "string" ||
      !samlResponse.trim() ||
      typeof relayState !== "string" ||
      !relayState.trim() ||
      !savedRelay ||
      !safeEqual(savedRelay, relayState)
    ) {
      return failRedirect(request.url);
    }

    // Forward SAMLResponse to Hermes /saml/acs for cryptographic assertion verification
    const hermesUrl = getServiceUrl("hermes");
    const acsRes = await fetch(`${hermesUrl}/saml/acs`, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        SAMLResponse: samlResponse,
        RelayState: relayState,
      }).toString(),
      cache: "no-store",
      signal: AbortSignal.timeout(SAML_VERIFY_TIMEOUT_MS),
    });

    if (!acsRes.ok) {
      return failRedirect(request.url);
    }

    const acsData = (await acsRes.json().catch(() => null)) as {
      status?: string;
      assertion?: {
        subject?: string;
        name_id?: string;
        email?: string;
        attributes?: Record<string, string | string[]>;
      };
    } | null;

    const assertion = acsData?.assertion;
    const rawEmail =
      assertion?.email ||
      (typeof assertion?.attributes?.email === "string"
        ? assertion.attributes.email
        : Array.isArray(assertion?.attributes?.email)
          ? assertion.attributes.email[0]
          : undefined) ||
      assertion?.name_id ||
      assertion?.subject;

    if (!rawEmail || typeof rawEmail !== "string") {
      return failRedirect(request.url);
    }

    // Exchange verified SAML identity with Argus for a real server-backed operator session
    const argusUrl = getServiceUrl("argus");
    const sessionHeaders: Record<string, string> = { "Content-Type": "application/json" };
    if (process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN) {
      sessionHeaders["x-autorix-console-token"] = process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN;
    }
    const sessionRes = await fetch(`${argusUrl}/v1/auth/login`, {
      method: "POST",
      headers: sessionHeaders,
      body: JSON.stringify({
        email: rawEmail,
        provider: "saml",
        assertion,
      }),
      cache: "no-store",
      signal: AbortSignal.timeout(SAML_VERIFY_TIMEOUT_MS),
    });

    if (!sessionRes.ok) {
      return failRedirect(request.url);
    }

    const sessionData = (await sessionRes.json().catch(() => null)) as {
      session_token?: string;
      expires_at?: string;
    } | null;

    if (!sessionData?.session_token || typeof sessionData.session_token !== "string") {
      return failRedirect(request.url);
    }

    const response = NextResponse.redirect(new URL("/", request.url));
    response.cookies.set({
      name: SESSION_COOKIE_NAME,
      value: sessionData.session_token,
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      ...(sessionData.expires_at ? { expires: new Date(sessionData.expires_at) } : { maxAge: 3600 * 12 }),
    });
    response.cookies.delete("saml_relay_state");

    return response;
  } catch {
    return failRedirect(request.url);
  }
}
