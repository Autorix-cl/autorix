import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { getServiceUrl } from "../api-config";
import { hasPermission, type Operator, type SessionResponse } from "./types";

export const SESSION_COOKIE_NAME = "autorix_session";
const SESSION_VALIDATION_TIMEOUT_MS = 10_000;
const SESSION_VALIDATION_MAX_ATTEMPTS = 2;
const SESSION_VALIDATION_RETRY_DELAY_MS = 200;

export class SessionValidationUnavailableError extends Error {
  constructor(message = "session validation backend temporarily unavailable") {
    super(message);
    this.name = "SessionValidationUnavailableError";
  }
}

export async function getSessionToken(): Promise<string | undefined> {
  const cookieStore = await cookies();
  return cookieStore.get(SESSION_COOKIE_NAME)?.value;
}

export async function validateSessionToken(token: string): Promise<{ valid: boolean; operator?: Operator }> {
  if (!token) return { valid: false };

  const argusUrl = getServiceUrl("argus");
  let lastError = "session validation backend unreachable";

  for (let attempt = 0; attempt < SESSION_VALIDATION_MAX_ATTEMPTS; attempt++) {
    try {
      const res = await fetch(`${argusUrl}/v1/auth/session`, {
        method: "GET",
        headers: {
          Authorization: `Bearer ${token}`,
        },
        cache: "no-store",
        signal: AbortSignal.timeout(SESSION_VALIDATION_TIMEOUT_MS),
      });

      if (res.status === 401 || res.status === 403) {
        return { valid: false };
      }

      if (!res.ok) {
        lastError = `argus session validation returned HTTP ${res.status}`;
      } else {
        const data: SessionResponse = await res.json();
        return { valid: data.valid, operator: data.operator };
      }
    } catch (err: unknown) {
      lastError = err instanceof Error ? err.message : "session validation request failed";
    }

    if (attempt + 1 < SESSION_VALIDATION_MAX_ATTEMPTS) {
      await new Promise((resolve) => setTimeout(resolve, SESSION_VALIDATION_RETRY_DELAY_MS));
    }
  }

  throw new SessionValidationUnavailableError(lastError);
}

export async function getCurrentOperator(): Promise<Operator | null> {
  const token = await getSessionToken();
  if (!token) return null;
  const { valid, operator } = await validateSessionToken(token);
  return valid && operator ? operator : null;
}

export type OperatorAuthResult = { ok: true; operator: Operator } | { ok: false; response: NextResponse };

export async function requireOperator(
  requiredPermission?: string,
  extraHeaders?: Record<string, string>,
): Promise<OperatorAuthResult> {
  let operator: Operator | null;
  try {
    operator = await getCurrentOperator();
  } catch (err) {
    if (err instanceof SessionValidationUnavailableError) {
      return {
        ok: false,
        response: NextResponse.json(
          { error: "service unavailable: session validation temporarily unavailable" },
          { status: 503, headers: extraHeaders },
        ),
      };
    }
    throw err;
  }

  if (!operator) {
    const response = NextResponse.json(
      { error: "unauthorized: authentication required" },
      { status: 401, headers: extraHeaders },
    );
    response.cookies.delete(SESSION_COOKIE_NAME);
    return { ok: false, response };
  }

  if (requiredPermission && !hasPermission(operator.role, requiredPermission)) {
    return {
      ok: false,
      response: NextResponse.json(
        {
          error: `forbidden: operator role '${operator.role}' lacks required permission '${requiredPermission}'`,
          required_permission: requiredPermission,
          role: operator.role,
        },
        { status: 403, headers: extraHeaders },
      ),
    };
  }

  return { ok: true, operator };
}
