/**
 * Typed API client with an explicit error taxonomy (P1-S3-T2).
 *
 * Every BFF route and every data-fetching call site goes through fetchJSON
 * instead of a bespoke `fetch().catch(() => [])`, so a stopped engine, a
 * network partition and a genuinely empty collection stop being
 * indistinguishable — each carries a distinct ApiErrorKind the UI can act on.
 */

/** The five failure modes a console request can land in. */
export type ApiErrorKind =
  | "engine-unreachable" // network failure, timeout, DNS — never got a response
  | "engine-error" // engine responded, but with a 5xx
  | "unauthorized" // 401/403
  | "validation" // 400/422 — the request itself was rejected
  | "unknown"; // any other status the taxonomy doesn't have a home for

export interface ApiError {
  kind: ApiErrorKind;
  message: string;
  status?: number;
  requestId?: string;
  cause?: unknown;
}

export type ApiResult<T> = { ok: true; data: T } | { ok: false; error: ApiError };

export interface FetchJSONOptions extends RequestInit {
  /** Aborts the request after this many milliseconds. Defaults to 10s. */
  timeoutMs?: number;
}

const DEFAULT_TIMEOUT_MS = 10_000;

function classifyStatus(status: number): ApiErrorKind {
  if (status === 401 || status === 403) return "unauthorized";
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

function getCsrfToken(): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(/(?:^|;\s*)autorix_csrf=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : null;
}

/**
 * Fetches url and returns a discriminated ApiResult instead of throwing —
 * callers branch on `result.ok` and never need a try/catch to tell a
 * network failure apart from a 4xx/5xx apart from success.
 */
export async function fetchJSON<T>(url: string, options: FetchJSONOptions = {}): Promise<ApiResult<T>> {
  const { timeoutMs = DEFAULT_TIMEOUT_MS, ...init } = options;
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  const headers = new Headers(init.headers);
  const method = (init.method || "GET").toUpperCase();
  if (["POST", "PUT", "PATCH", "DELETE"].includes(method) && !headers.has("X-CSRF-Token")) {
    const csrf = getCsrfToken();
    if (csrf) {
      headers.set("X-CSRF-Token", csrf);
    }
  }

  let res: Response;
  try {
    res = await fetch(url, { ...init, headers, signal: controller.signal });
  } catch (err) {
    clearTimeout(timer);
    if (err instanceof DOMException && err.name === "AbortError") {
      return { ok: false, error: { kind: "engine-unreachable", message: "request timed out", cause: err } };
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
