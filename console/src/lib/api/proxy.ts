/**
 * Shared server-side proxy for Next.js route handlers (P1-S3-T3, P3-S4, P3-S5).
 * Enforces server-side RBAC, SSRF protection over registered engine backends,
 * error taxonomy classification and response schema validation.
 */
import { NextResponse } from "next/server";
import { randomUUID } from "crypto";
import type { z } from "zod";
import { getServiceUrl, BACKEND_URLS, JANUS_ADMIN_URL } from "../api-config";
import { getCurrentOperator, SESSION_COOKIE_NAME } from "../auth/session";
import { hasPermission } from "../auth/types";

type Service = keyof typeof BACKEND_URLS;

const UPSTREAM_TIMEOUT_MS = 15_000;

const SERVICE_PERMISSION_DOMAINS: Record<Service, string> = {
  argus: "fleet",
  ego: "identities",
  janus: "oauth2",
  aegis: "proxy-rules",
  nexus: "permissions",
  themis: "policies",
  vulcan: "api-keys",
  hermes: "enterprise",
  prometheus: "observability",
};

export interface ProxyOptions extends RequestInit {
  requiredPermission?: string;
}

function extractMessage(body: unknown, rawText: string, status: number): string {
  if (body && typeof body === "object" && "error" in body && typeof (body as { error: unknown }).error === "string") {
    return (body as { error: string }).error;
  }
  if (rawText.trim()) return rawText;
  return `upstream request failed with status ${status}`;
}

function isValidUpstreamPath(rawPath: string): boolean {
  if (!rawPath || !rawPath.startsWith("/") || rawPath.startsWith("//")) {
    return false;
  }
  if (rawPath.includes("\\") || rawPath.includes("\0")) {
    return false;
  }
  const pathOnly = rawPath.split(/[?#]/, 1)[0];
  const segments = pathOnly.split("/");
  for (const seg of segments) {
    if (seg === "..") {
      return false;
    }
    let decoded = seg;
    for (let pass = 0; pass < 2; pass++) {
      try {
        const next = decodeURIComponent(decoded);
        if (next === decoded) break;
        decoded = next;
      } catch {
        return false;
      }
    }
    if (
      decoded === ".." ||
      decoded.includes("..") ||
      decoded.includes("/") ||
      decoded.includes("\\") ||
      decoded.includes("\0")
    ) {
      return false;
    }
  }
  return true;
}

/**
 * Fetches path on service, enforces RBAC, validates body against schema, and returns
 * a NextResponse.
 */
export async function proxyRequest<T>(
  service: Service,
  path: string,
  schema: z.ZodType<T>,
  options?: ProxyOptions,
): Promise<NextResponse> {
  const requestId = randomUUID();
  const headers = { "x-request-id": requestId, "x-correlation-id": requestId };

  // 1. SSRF Protection (P3-S5-T1): reject unknown service destinations
  if (!BACKEND_URLS[service]) {
    return NextResponse.json(
      { error: `forbidden: destination service ${service} is not a valid registered engine` },
      { status: 403, headers },
    );
  }

  // 2. Path traversal protection: reject malformed or escaping upstream paths
  if (!isValidUpstreamPath(path)) {
    return NextResponse.json(
      { error: "bad request: invalid upstream path" },
      { status: 400, headers },
    );
  }

  // 3. Server-side Authentication & RBAC Enforcement (P3-S4-T2)
  // Every proxied engine call requires a validated operator session and baseline domain permission.
  const operator = await getCurrentOperator();
  if (!operator) {
    const res = NextResponse.json({ error: "unauthorized: authentication required" }, { status: 401, headers });
    res.cookies.delete(SESSION_COOKIE_NAME);
    return res;
  }

  const permissionDomain = SERVICE_PERMISSION_DOMAINS[service];
  const isReadMethod = ["GET", "HEAD"].includes((options?.method || "GET").toUpperCase());
  const baselineAction = isReadMethod ? "read" : permissionDomain === "fleet" ? "manage" : "write";
  const baselinePermission = `${permissionDomain}:${baselineAction}`;
  const requiredPermissions = [baselinePermission, options?.requiredPermission].filter(
    (permission): permission is string => Boolean(permission),
  );

  const missingPermission = requiredPermissions.find((permission) => !hasPermission(operator.role, permission));
  if (missingPermission) {
    return NextResponse.json(
      {
        error: `forbidden: operator role '${operator.role}' lacks required permission '${missingPermission}'`,
        required_permission: missingPermission,
        role: operator.role,
      },
      { status: 403, headers },
    );
  }

  // Public OAuth endpoints stay on the public listener; management stays private.
  const baseUrl = service === "janus" && path.startsWith("/admin/") ? JANUS_ADMIN_URL : getServiceUrl(service);
  const url = `${baseUrl}${path}`;

  let res: Response;
  try {
    const init = options ? { ...options } : undefined;
    if (init) delete init.requiredPermission;

    const requestHeaders = new Headers(init?.headers);
    requestHeaders.set("x-request-id", requestId);
    requestHeaders.set("x-correlation-id", requestId);
    requestHeaders.set("x-autorix-caller", "console");
    const internalToken = process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN;
    if (internalToken) {
      requestHeaders.set("x-autorix-console-token", internalToken);
    }

    const timeoutSignal = AbortSignal.timeout(UPSTREAM_TIMEOUT_MS);
    const signal = init?.signal ? AbortSignal.any([init.signal, timeoutSignal]) : timeoutSignal;

    res = await fetch(url, {
      cache: "no-store",
      ...init,
      signal,
      headers: requestHeaders,
    });
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? `${service} is unreachable: ${err.message}` : `${service} is unreachable` },
      { status: 502, headers },
    );
  }

  const rawText = await res.text();
  let body: unknown = null;
  if (rawText) {
    try {
      body = JSON.parse(rawText);
    } catch {
      // Non-JSON upstream body; message extraction below falls back to raw text.
    }
  }

  if (!res.ok) {
    return NextResponse.json({ error: extractMessage(body, rawText, res.status) }, { status: res.status, headers });
  }

  if (res.status === 204) {
    return new NextResponse(null, { status: 204, headers });
  }

  const parsed = schema.safeParse(body);
  if (!parsed.success) {
    const message = parsed.error.issues
      .map((issue) => `${issue.path.join(".") || "(root)"}: ${issue.message}`)
      .join("; ");
    return NextResponse.json(
      { error: `unexpected response shape from ${service}: ${message}` },
      { status: 502, headers },
    );
  }

  return NextResponse.json(parsed.data, { status: res.status, headers });
}
