/**
 * Typed API client with an explicit error taxonomy (P1-S3-T2).
 *
 * Every BFF route and every data-fetching call site goes through fetchJSON
 * instead of a bespoke `fetch().catch(() => [])`, so a stopped engine, a
 * network partition and a genuinely empty collection stop being
 * indistinguishable — each carries a distinct ApiErrorKind the UI can act on.
 */

/** The six failure modes a console request can land in. */
export type ApiErrorKind =
  | "engine-unreachable" // network failure, timeout, DNS — never got a response
  | "engine-error" // engine responded, but with a 5xx
  | "unauthorized" // 401 — unauthenticated / expired session
  | "forbidden" // 403 — authenticated, but lacks RBAC permission or CSRF
  | "validation" // 400/422 — the request itself was rejected
  | "unknown"; // any other status the taxonomy doesn't have a home for

export interface ApiError {
  kind: ApiErrorKind;
  message: string;
  status?: number;
  requestId?: string;
  cause?: unknown;
}

export class ApiRequestError extends Error {
  readonly apiError: ApiError;
  readonly kind: ApiErrorKind;
  readonly status?: number;
  readonly requestId?: string;

  constructor(apiError: ApiError) {
    super(apiError.message);
    this.name = "ApiRequestError";
    this.apiError = apiError;
    this.kind = apiError.kind;
    this.status = apiError.status;
    this.requestId = apiError.requestId;
  }
}

export type ApiResult<T> = { ok: true; data: T } | { ok: false; error: ApiError };

export interface FetchJSONOptions extends RequestInit {
  /** Aborts the request after this many milliseconds. Defaults to 10s. */
  timeoutMs?: number;
}

const DEFAULT_TIMEOUT_MS = 10_000;

function classifyStatus(status: number): ApiErrorKind {
  if (status === 401) return "unauthorized";
  if (status === 403) return "forbidden";
  if (status === 400 || status === 422) return "validation";
  if (status >= 500) return "engine-error";
  return "unknown";
}

function extractMessage(body: unknown, rawText: string, status: number): string {
  if (body && typeof body === "object" && "error" in body && typeof (body as { error: unknown }).error === "string") {
    return (body as { error: string }).error;
  }
  if (rawText.trim()) return rawText;
  return `request failed with status ${status}`;
}

let isRedirecting = false;
let redirectResetTimer: ReturnType<typeof setTimeout> | null = null;
let redirectHandler: ((targetUrl: string) => void) | null = null;

export function setRedirectHandler(handler: ((targetUrl: string) => void) | null): void {
  redirectHandler = handler;
}

export function redirectToLogin(fromPath?: string): void {
  if (typeof window === "undefined" || isRedirecting) return;

  const pathname = fromPath || window.location.pathname;
  if (
    pathname === "/login" ||
    pathname.startsWith("/login/") ||
    pathname === "/setup" ||
    pathname.startsWith("/setup/") ||
    pathname === "/403" ||
    pathname === "/session-expired"
  ) {
    return;
  }

  isRedirecting = true;
  const from = encodeURIComponent(pathname + (window.location.search || ""));
  const target = `/login?from=${from}`;

  if (redirectHandler) {
    redirectHandler(target);
    return;
  }

  // Guard against JSDOM / Vitest unit tests where navigation is unsupported
  if (process.env.NODE_ENV === "test") return;

  // Failsafe reset if navigation is cancelled or blocked by the browser
  redirectResetTimer = setTimeout(() => {
    isRedirecting = false;
    redirectResetTimer = null;
  }, 5_000);

  try {
    window.location.href = target;
  } catch {
    isRedirecting = false;
    if (redirectResetTimer) {
      clearTimeout(redirectResetTimer);
      redirectResetTimer = null;
    }
  }
}

export function resetRedirectFlag(): void {
  isRedirecting = false;
  redirectHandler = null;
  if (redirectResetTimer) {
    clearTimeout(redirectResetTimer);
    redirectResetTimer = null;
  }
}

function getCsrfToken(): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(/(?:^|;\s*)autorix_csrf=([^;]*)/);
  if (!match) return null;
  try {
    return decodeURIComponent(match[1]);
  } catch {
    return match[1];
  }
}

export function getCsrfHeaders(): Record<string, string> {
  const token = getCsrfToken();
  return token ? { "X-CSRF-Token": token } : {};
}

/**
 * Fetches url and returns a discriminated ApiResult instead of throwing —
 * callers branch on `result.ok` and never need a try/catch to tell a
 * network failure apart from a 4xx/5xx apart from success.
 */
export async function fetchJSON<T>(url: string, options: FetchJSONOptions = {}): Promise<ApiResult<T>> {
  const { timeoutMs = DEFAULT_TIMEOUT_MS, ...init } = options;
  const controller = new AbortController();
  const callerSignal = init.signal;
  let timedOut = false;

  const onCallerAbort = () => controller.abort(callerSignal?.reason);
  if (callerSignal) {
    if (callerSignal.aborted) {
      controller.abort(callerSignal.reason);
    } else {
      callerSignal.addEventListener("abort", onCallerAbort, { once: true });
    }
  }

  const timer = setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, timeoutMs);

  const headers = new Headers(init.headers);
  const method = (init.method || "GET").toUpperCase();
  if (["POST", "PUT", "PATCH", "DELETE"].includes(method) && !headers.has("X-CSRF-Token")) {
    const csrfHeaders = getCsrfHeaders();
    if (csrfHeaders["X-CSRF-Token"]) {
      headers.set("X-CSRF-Token", csrfHeaders["X-CSRF-Token"]);
    }
  }

  let res: Response;
  try {
    res = await fetch(url, { ...init, headers, signal: controller.signal });
  } catch (err) {
    clearTimeout(timer);
    callerSignal?.removeEventListener("abort", onCallerAbort);
    if (err instanceof DOMException && err.name === "AbortError") {
      return {
        ok: false,
        error: {
          kind: "engine-unreachable",
          message: timedOut ? "request timed out" : "request aborted",
          cause: err,
        },
      };
    }
    return {
      ok: false,
      error: {
        kind: "engine-unreachable",
        message: err instanceof Error ? err.message : "network error",
        cause: err,
      },
    };
  }
  clearTimeout(timer);
  callerSignal?.removeEventListener("abort", onCallerAbort);

  const requestId = res.headers.get("x-request-id") ?? undefined;
  const rawText = await res.text();
  let body: unknown = null;
  if (rawText) {
    try {
      body = JSON.parse(rawText);
    } catch {
      // Non-JSON body (e.g. an upstream proxy error page); message falls
      // back to the raw text below.
    }
  }

  if (!res.ok) {
    if (res.status === 401 && typeof window !== "undefined") {
      const isPublicAuthRoute =
        url.includes("/api/auth/login") ||
        url.includes("/api/auth/status") ||
        url.includes("/api/auth/setup") ||
        url.includes("/api/auth/sso/") ||
        url.includes("/api/auth/saml/");
      if (!isPublicAuthRoute) {
        redirectToLogin();
      }
    }

    return {
      ok: false,
      error: {
        kind: classifyStatus(res.status),
        message: extractMessage(body, rawText, res.status),
        status: res.status,
        requestId,
        cause: body ?? rawText,
      },
    };
  }

  return { ok: true, data: body as T };
}
