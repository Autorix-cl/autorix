import { createHash } from "crypto";
import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { getServiceUrl } from "../api-config";
import { hasPermission, type Operator, type SessionResponse } from "./types";

export const SESSION_COOKIE_NAME = "autorix_session";
const SESSION_VALIDATION_TIMEOUT_MS = 10_000;
const SESSION_VALIDATION_MAX_ATTEMPTS = 3;
const SESSION_VALIDATION_RETRY_DELAY_MS = 250;
const SESSION_CACHE_TTL_MS = 15_000;
const SESSION_CACHE_MAX_ENTRIES = 256;

interface CachedSession {
  operator: Operator;
  expiresAt: number;
}

type SessionValidationResult = { valid: boolean; operator?: Operator };

// Store cache and single-flight map on globalThis so separate Next.js route
// chunks share one validation state and logout invalidation is server-wide.
const globalForSession = globalThis as typeof globalThis & {
  __autorixSessionCache?: Map<string, CachedSession>;
  __autorixInFlightValidations?: Map<string, Promise<SessionValidationResult>>;
};

const sessionCache = (globalForSession.__autorixSessionCache ??= new Map<string, CachedSession>());
const inFlightValidations = (globalForSession.__autorixInFlightValidations ??= new Map<
  string,
  Promise<SessionValidationResult>
>());

function hashSessionToken(token: string): string {
  return createHash("sha256").update(token).digest("hex");
}

export function invalidateSessionCache(token?: string): void {
  if (!token) return;
  sessionCache.delete(hashSessionToken(token));
}

export function clearSessionCache(): void {
  sessionCache.clear();
  inFlightValidations.clear();
}

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

async function fetchSessionValidation(
  token: string,
  tokenKey: string,
): Promise<{ valid: boolean; operator?: Operator }> {
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
        sessionCache.delete(tokenKey);
        return { valid: false };
      }

      if (!res.ok) {
        lastError = `argus session validation returned HTTP ${res.status}`;
      } else {
        const data: SessionResponse = await res.json();
        if (data.valid && data.operator) {
          if (sessionCache.size >= SESSION_CACHE_MAX_ENTRIES) {
            const oldestKey = sessionCache.keys().next().value;
            if (oldestKey) sessionCache.delete(oldestKey);
          }
          sessionCache.set(tokenKey, {
            operator: data.operator,
            expiresAt: Date.now() + SESSION_CACHE_TTL_MS,
          });
        } else {
          sessionCache.delete(tokenKey);
        }
        return { valid: data.valid, operator: data.operator };
      }
    } catch (err: unknown) {
      lastError = err instanceof Error ? err.message : "session validation request failed";
    }

    if (attempt + 1 < SESSION_VALIDATION_MAX_ATTEMPTS) {
      await new Promise((resolve) => setTimeout(resolve, SESSION_VALIDATION_RETRY_DELAY_MS * (attempt + 1)));
    }
  }

  console.error("[session] validation unavailable after retries:", lastError);
  throw new SessionValidationUnavailableError(lastError);
}

export async function validateSessionToken(token: string): Promise<{ valid: boolean; operator?: Operator }> {
  if (!token) return { valid: false };

  const tokenKey = hashSessionToken(token);
  const cached = sessionCache.get(tokenKey);
  if (cached) {
    if (cached.expiresAt > Date.now()) {
      return { valid: true, operator: cached.operator };
    }
    sessionCache.delete(tokenKey);
  }

  const pending = inFlightValidations.get(tokenKey);
  if (pending) {
    return pending;
  }

  const request = fetchSessionValidation(token, tokenKey).finally(() => {
    inFlightValidations.delete(tokenKey);
  });
  inFlightValidations.set(tokenKey, request);
  return request;
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
