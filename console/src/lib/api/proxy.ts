/**
 * Shared server-side proxy for Next.js route handlers (P1-S3-T3, P3-S4, P3-S5).
 * Enforces server-side RBAC, SSRF protection over registered engine backends,
 * error taxonomy classification and response schema validation.
 */
import { NextResponse } from "next/server";
import { randomUUID } from "crypto";
import type { z } from "zod";
import { getServiceUrl, BACKEND_URLS } from "../api-config";
import { getCurrentOperator } from "../auth/session";
import { hasPermission } from "../auth/types";

type Service = keyof typeof BACKEND_URLS;

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
  const headers = { "x-request-id": requestId };

  // 1. SSRF Protection (P3-S5-T1): reject unknown service destinations
  if (!BACKEND_URLS[service]) {
    return NextResponse.json(
      { error: `forbidden: destination service ${service} is not a valid registered engine` },
      { status: 403, headers },
    );
  }

  // 2. Server-side RBAC Enforcement (P3-S4-T2)
  if (options?.requiredPermission) {
    const operator = await getCurrentOperator();
    if (!operator) {
      return NextResponse.json(
        { error: "unauthorized: authentication required" },
        { status: 401, headers },
      );
    }

    if (!hasPermission(operator.role, options.requiredPermission)) {
      return NextResponse.json(
        {
          error: `forbidden: operator role '${operator.role}' lacks required permission '${options.requiredPermission}'`,
          required_permission: options.requiredPermission,
          role: operator.role,
        },
        { status: 403, headers },
      );
    }
  }

  const url = `${getServiceUrl(service)}${path}`;

  let res: Response;
  try {
    const init = options ? { ...options } : undefined;
    if (init) delete init.requiredPermission;

    const requestHeaders = new Headers(init?.headers);
    requestHeaders.set("x-request-id", requestId);
    requestHeaders.set("x-autorix-caller", "console");
    requestHeaders.set("x-autorix-console-token", "act_internal_console_trusted");

    res = await fetch(url, {
      cache: "no-store",
      ...init,
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
