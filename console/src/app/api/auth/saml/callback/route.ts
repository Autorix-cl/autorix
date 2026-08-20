import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { SESSION_COOKIE_NAME } from "@/lib/auth/session";

export async function POST(request: Request) {
  try {
    const formData = await request.formData();
    const samlResponse = formData.get("SAMLResponse") as string | null;
    const relayState = formData.get("RelayState") as string | null;

    const cookieStore = await cookies();
    const savedRelay = cookieStore.get("saml_relay_state")?.value;

    if (!samlResponse || (savedRelay && relayState && savedRelay !== relayState)) {
      return NextResponse.redirect(new URL("/login?error=invalid_saml_response", request.url));
    }

    // In a full Hermes integration, the SAML XML assertion is validated against Hermes SP cert
    const sessionToken = "ast_saml_" + crypto.randomUUID().replace(/-/g, "");

    const response = NextResponse.redirect(new URL("/", request.url));
    response.cookies.set({
      name: SESSION_COOKIE_NAME,
      value: sessionToken,
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: 3600 * 12,
    });
    response.cookies.delete("saml_relay_state");

    return response;
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : "SAML authorization failed";
    return NextResponse.redirect(new URL(`/login?error=${encodeURIComponent(message)}`, request.url));
  }
}
