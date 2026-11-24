import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { timingSafeEqual } from "crypto";
import { getJanusDiscovery } from "@/lib/auth/sso";
import { SESSION_COOKIE_NAME } from "@/lib/auth/session";
import { getServiceUrl } from "@/lib/api-config";

const SSO_CALLBACK_TIMEOUT_MS = 10_000;

function safeEqual(a: string, b: string): boolean {
  const bufA = Buffer.from(a, "utf8");
  const bufB = Buffer.from(b, "utf8");
  if (bufA.length !== bufB.length) return false;
  return timingSafeEqual(bufA, bufB);
}

function failRedirect(requestUrl: string): NextResponse {
  const response = NextResponse.redirect(new URL("/login?error=sso_failed", requestUrl));
  response.cookies.delete("sso_state");
  response.cookies.delete("sso_verifier");
  return response;
}

interface VerifiedClaims {
  email?: string;
  sub?: string;
  name?: string;
  role?: string;
}

async function verifyTokenWithJanus(
  janusUrl: string,
  accessToken: string,
  userinfoEndpoint?: string,
  introspectionEndpoint?: string,
): Promise<VerifiedClaims | null> {
  const userinfoUrl = userinfoEndpoint || `${janusUrl}/oauth2/userinfo`;
  const userinfoRes = await fetch(userinfoUrl, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${accessToken}`,
      Accept: "application/json",
    },
    cache: "no-store",
    signal: AbortSignal.timeout(SSO_CALLBACK_TIMEOUT_MS),
  });

  if (userinfoRes.ok) {
    const claims = (await userinfoRes.json().catch(() => null)) as VerifiedClaims | null;
    if (claims && (claims.email || claims.sub)) {
      return claims;
    }
    return null;
  }

  // Fallback to OAuth2 token introspection if userinfo is not exposed on this Janus build
  if (userinfoRes.status === 404 || userinfoRes.status === 405) {
    const introspectUrl = introspectionEndpoint || `${janusUrl}/oauth2/introspect`;
    const introspectBody = new URLSearchParams({ token: accessToken });
    const introspectHeaders: Record<string, string> = {
      "Content-Type": "application/x-www-form-urlencoded",
    };
    const clientSecret = process.env.JANUS_CONSOLE_CLIENT_SECRET;
    if (clientSecret) {
      const basic = Buffer.from(`autorix-console:${clientSecret}`).toString("base64");
      introspectHeaders.Authorization = `Basic ${basic}`;
    }

    const introspectRes = await fetch(introspectUrl, {
      method: "POST",
      headers: introspectHeaders,
      body: introspectBody.toString(),
      cache: "no-store",
      signal: AbortSignal.timeout(SSO_CALLBACK_TIMEOUT_MS),
    });

    if (!introspectRes.ok) return null;
    const introspectData = (await introspectRes.json().catch(() => null)) as (VerifiedClaims & {
      active?: boolean;
    }) | null;
    if (!introspectData?.active) return null;
    if (introspectData.email || introspectData.sub) {
      return introspectData;
    }
  }

  return null;
}

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const code = searchParams.get("code");
  const state = searchParams.get("state");
  const error = searchParams.get("error");

  if (error || !code || !state) {
    return failRedirect(request.url);
  }

  const cookieStore = await cookies();
  const savedState = cookieStore.get("sso_state")?.value;
  const verifier = cookieStore.get("sso_verifier")?.value;

  if (!savedState || !verifier || !safeEqual(savedState, state)) {
    return failRedirect(request.url);
  }

  try {
    const discovery = await getJanusDiscovery();
    if (!discovery) {
      return failRedirect(request.url);
    }

    const janusUrl = getServiceUrl("janus").replace(/\/+$/, "");
    const redirectUri = new URL("/api/auth/sso/callback", request.url).toString();
    const tokenEndpoint = `${janusUrl}/oauth2/token`;

    const tokenRes = await fetch(tokenEndpoint, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        grant_type: "authorization_code",
        client_id: "autorix-console",
        code,
        redirect_uri: redirectUri,
        code_verifier: verifier,
      }).toString(),
      cache: "no-store",
      signal: AbortSignal.timeout(SSO_CALLBACK_TIMEOUT_MS),
    });

    if (!tokenRes.ok) {
      return failRedirect(request.url);
    }

    const tokenData = (await tokenRes.json().catch(() => null)) as {
      access_token?: string;
    } | null;

    if (!tokenData?.access_token) {
      return failRedirect(request.url);
    }

    // Verify token & retrieve claims directly from Janus rather than trusting unverified JWT base64 decode
    const claims = await verifyTokenWithJanus(
      janusUrl,
      tokenData.access_token,
      discovery.userinfo_endpoint,
      discovery.introspection_endpoint,
    );
    if (!claims || (!claims.email && !claims.sub)) {
      return failRedirect(request.url);
    }

    const email = claims.email || claims.sub!;
    const name = claims.name || email;
    const role =
      claims.role && ["owner", "admin", "operator", "auditor"].includes(claims.role)
        ? claims.role
        : "operator";

    // Exchange verified federated identity for a real Argus operator session; never mint a fake fallback cookie
    const argusUrl = getServiceUrl("argus");
    const sessionHeaders: Record<string, string> = { "Content-Type": "application/json" };
    if (process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN) {
      sessionHeaders["x-autorix-console-token"] = process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN;
    }
    const sessionRes = await fetch(`${argusUrl}/v1/auth/login`, {
      method: "POST",
      headers: sessionHeaders,
      body: JSON.stringify({ email, name, role, provider: "oidc" }),
      cache: "no-store",
      signal: AbortSignal.timeout(SSO_CALLBACK_TIMEOUT_MS),
    });

    if (!sessionRes.ok) {
      return failRedirect(request.url);
    }

    const ssoSession = (await sessionRes.json().catch(() => null)) as {
      session_token?: string;
      expires_at?: string;
    } | null;

    if (!ssoSession?.session_token || typeof ssoSession.session_token !== "string") {
      return failRedirect(request.url);
    }

    const response = NextResponse.redirect(new URL("/", request.url));
    response.cookies.set({
      name: SESSION_COOKIE_NAME,
      value: ssoSession.session_token,
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      ...(ssoSession.expires_at
        ? { expires: new Date(ssoSession.expires_at) }
        : { maxAge: 3600 * 12 }),
    });

    response.cookies.delete("sso_state");
    response.cookies.delete("sso_verifier");

    return response;
  } catch {
    return failRedirect(request.url);
  }
}
