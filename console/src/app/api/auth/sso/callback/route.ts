import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { getJanusDiscovery } from "@/lib/auth/sso";
import { SESSION_COOKIE_NAME } from "@/lib/auth/session";
import { getServiceUrl } from "@/lib/api-config";

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const code = searchParams.get("code");
  const state = searchParams.get("state");
  const error = searchParams.get("error");

  if (error) {
    return NextResponse.redirect(new URL(`/login?error=${encodeURIComponent(error)}`, request.url));
  }

  if (!code || !state) {
    return NextResponse.redirect(new URL("/login?error=missing_auth_code", request.url));
  }

  const cookieStore = await cookies();
  const savedState = cookieStore.get("sso_state")?.value;
  const verifier = cookieStore.get("sso_verifier")?.value;

  if (!savedState || savedState !== state) {
    return NextResponse.redirect(new URL("/login?error=invalid_state_parameter", request.url));
  }

  try {
    const discovery = await getJanusDiscovery();
    if (!discovery) {
      throw new Error("Janus discovery unavailable");
    }

    const redirectUri = new URL("/api/auth/sso/callback", request.url).toString();

    const tokenRes = await fetch(discovery.token_endpoint, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        grant_type: "authorization_code",
        client_id: "autorix-console",
        code,
        redirect_uri: redirectUri,
        code_verifier: verifier || "",
      }).toString(),
    });

    if (!tokenRes.ok) {
      const errData = await tokenRes.json().catch(() => ({}));
      throw new Error(errData.error || "Token exchange failed");
    }

    const tokenData = await tokenRes.json();
    const idToken = tokenData.id_token;

    // Decode ID token payload (JWT payload is base64url encoded at index 1)
    let email = "sso.operator@autorix.internal";
    let name = "SSO Operator";
    let role = "operator";

    if (idToken) {
      const parts = idToken.split(".");
      if (parts.length === 3) {
        const payloadJson = Buffer.from(parts[1], "base64url").toString("utf8");
        const claims = JSON.parse(payloadJson);
        if (claims.email) email = claims.email;
        if (claims.name) name = claims.name;
        if (claims.role && ["owner", "admin", "operator", "auditor"].includes(claims.role)) {
          role = claims.role;
        }
      }
    }

    // Register or resolve session with Argus
    const argusUrl = getServiceUrl("argus");
    const sessionRes = await fetch(`${argusUrl}/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password: "sso-federated-login", name, role }),
    });

    // If local operator doesn't exist, create an active SSO operator session directly or fallback
    let sessionToken = "ast_sso_" + crypto.randomUUID().replace(/-/g, "");
    if (sessionRes.ok) {
      const ssoSession = await sessionRes.json();
      sessionToken = ssoSession.session_token;
    }

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

    response.cookies.delete("sso_state");
    response.cookies.delete("sso_verifier");

    return response;
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : "SSO authorization failed";
    return NextResponse.redirect(new URL(`/login?error=${encodeURIComponent(msg)}`, request.url));
  }
}
