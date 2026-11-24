import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const SESSION_COOKIE_NAME = "autorix_session";
const CSRF_COOKIE_NAME = "autorix_csrf";
const STATIC_ASSET_EXT_RE = /\.(?:ico|png|jpg|jpeg|gif|svg|webp|woff2?|ttf|css|js|map)$/i;

// Explicit public allowlist (P3-S2-T1)
const PUBLIC_PATHS = ["/login", "/setup", "/403", "/api/health"];

const PUBLIC_API_PATHS = [
  "/api/auth/login",
  "/api/auth/setup",
  "/api/auth/status",
  "/api/auth/sso/login",
  "/api/auth/sso/callback",
  "/api/auth/saml/login",
  "/api/auth/saml/callback",
];

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const method = request.method.toUpperCase();

  // Ignore static assets and Next.js internal files, never bypassing /api/ routes
  const isStaticAsset =
    pathname.startsWith("/_next") ||
    pathname.startsWith("/static") ||
    (!pathname.startsWith("/api/") && STATIC_ASSET_EXT_RE.test(pathname));
  if (isStaticAsset) {
    return NextResponse.next();
  }

  const isPublicPage = PUBLIC_PATHS.some((path) => pathname === path || pathname.startsWith(path + "/"));
  const isPublicApi = PUBLIC_API_PATHS.some((path) => pathname === path || pathname.startsWith(path + "/"));

  const sessionCookie = request.cookies.get(SESSION_COOKIE_NAME)?.value;

  // If user is accessing public auth pages but already has a session, redirect to root
  if (sessionCookie && (pathname === "/login" || pathname === "/setup")) {
    return NextResponse.redirect(new URL("/", request.url));
  }

  // If public route or API, allow without session
  if (isPublicPage || isPublicApi) {
    const response = NextResponse.next();
    ensureCsrfCookie(request, response);
    return response;
  }

  // Protected route and no session cookie: reject API calls or redirect browser to /login
  if (!sessionCookie) {
    if (pathname.startsWith("/api/")) {
      return NextResponse.json({ error: "unauthorized: authentication required" }, { status: 401 });
    }
    const loginUrl = new URL("/login", request.url);
    if (pathname !== "/") {
      loginUrl.searchParams.set("from", pathname);
    }
    return NextResponse.redirect(loginUrl);
  }

  // Double-submit CSRF protection on mutating protected BFF routes (P3-S2-T3)
  const isMutatingApi = pathname.startsWith("/api/") && ["POST", "PUT", "PATCH", "DELETE"].includes(method);
  if (isMutatingApi) {
    const csrfCookie = request.cookies.get(CSRF_COOKIE_NAME)?.value;
    const csrfHeader = request.headers.get("X-CSRF-Token");
    const origin = request.headers.get("origin");

    const isSameOrigin = isTrustedSameOrigin(origin, request.nextUrl.origin);
    const hasValidToken = Boolean(csrfCookie && csrfHeader && constantTimeEqual(csrfCookie, csrfHeader));

    if (!isSameOrigin && !hasValidToken) {
      return NextResponse.json({ error: "forbidden: csrf verification failed" }, { status: 403 });
    }
  }

  const response = NextResponse.next();
  ensureCsrfCookie(request, response);
  return response;
}

function constantTimeEqual(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  let diff = 0;
  for (let i = 0; i < a.length; i++) {
    diff |= a.charCodeAt(i) ^ b.charCodeAt(i);
  }
  return diff === 0;
}

// Origin is a URL, not a substring. A check such as origin.includes(host)
// accepts attacker-controlled lookalikes (for example console.example.evil).
function isTrustedSameOrigin(origin: string | null, expectedOrigin: string): boolean {
  if (!origin) return false;
  try {
    return new URL(origin).origin === expectedOrigin;
  } catch {
    return false;
  }
}

function ensureCsrfCookie(request: NextRequest, response: NextResponse) {
  if (!request.cookies.has(CSRF_COOKIE_NAME)) {
    const randomBytes = new Uint8Array(16);
    crypto.getRandomValues(randomBytes);
    const token = Array.from(randomBytes)
      .map((b) => b.toString(16).padStart(2, "0"))
      .join("");

    response.cookies.set({
      name: CSRF_COOKIE_NAME,
      value: token,
      httpOnly: false, // Must be readable by client JS to set X-CSRF-Token header
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
    });
  }
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
