import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const SESSION_COOKIE_NAME = "autorix_session";
const CSRF_COOKIE_NAME = "autorix_csrf";

// Explicit public allowlist (P3-S2-T1)
const PUBLIC_PATHS = [
  "/login",
  "/login/mfa",
  "/setup",
  "/session-expired",
  "/403",
  "/health/alive",
  "/health/ready",
  "/info",
];

const PUBLIC_API_PATHS = [
  "/api/auth/login",
  "/api/auth/setup",
  "/api/auth/status",
  "/api/auth/logout",
];

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const method = request.method.toUpperCase();

  // Ignore static assets, next internal files, and favicon
  if (
    pathname.startsWith("/_next") ||
    pathname.startsWith("/static") ||
    pathname.includes(".") // file extensions like .ico, .png, .svg
  ) {
    return NextResponse.next();
  }

  const isPublicPage = PUBLIC_PATHS.some((path) => pathname === path || pathname.startsWith(path + "/"));
  const isPublicApi = PUBLIC_API_PATHS.some((path) => pathname === path || pathname.startsWith(path + "/"));

  const sessionCookie = request.cookies.get(SESSION_COOKIE_NAME)?.value;

  // If user is accessing public auth pages but already has a session, let them proceed or redirect to root
  if (sessionCookie && (pathname === "/login" || pathname === "/setup")) {
    return NextResponse.redirect(new URL("/", request.url));
  }

  // If public route or API, allow without session
  if (isPublicPage || isPublicApi) {
    const response = NextResponse.next();
    ensureCsrfCookie(request, response);
    return response;
  }

  // Protected route and no session cookie: redirect to /login
  if (!sessionCookie) {
    if (pathname.startsWith("/api/")) {
      return NextResponse.json(
        { error: "unauthorized: authentication required" },
        { status: 401 }
      );
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
    if (!csrfCookie || !csrfHeader || csrfCookie !== csrfHeader) {
      return NextResponse.json(
        { error: "forbidden: csrf verification failed" },
        { status: 403 }
      );
    }
  }

  const response = NextResponse.next();
  ensureCsrfCookie(request, response);
  return response;
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
  matcher: [
    "/((?!_next/static|_next/image|favicon.ico).*)",
  ],
};
