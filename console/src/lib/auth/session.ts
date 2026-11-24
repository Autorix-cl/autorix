import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { getServiceUrl } from "../api-config";
import { hasPermission, type Operator, type SessionResponse } from "./types";

export const SESSION_COOKIE_NAME = "autorix_session";
const SESSION_VALIDATION_TIMEOUT_MS = 8_000;

export async function getSessionToken(): Promise<string | undefined> {
  const cookieStore = await cookies();
  return cookieStore.get(SESSION_COOKIE_NAME)?.value;
}

export async function validateSessionToken(token: string): Promise<{ valid: boolean; operator?: Operator }> {
  if (!token) return { valid: false };

  try {
    const argusUrl = getServiceUrl("argus");
    const res = await fetch(`${argusUrl}/v1/auth/session`, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${token}`,
      },
      cache: "no-store",
      signal: AbortSignal.timeout(SESSION_VALIDATION_TIMEOUT_MS),
    });

    if (!res.ok) {
      return { valid: false };
    }

    const data: SessionResponse = await res.json();
    return { valid: data.valid, operator: data.operator };
  } catch {
    return { valid: false };
  }
}

export async function getCurrentOperator(): Promise<Operator | null> {
  const token = await getSessionToken();
  if (!token) return null;
  const { valid, operator } = await validateSessionToken(token);
  return valid && operator ? operator : null;
}

export type OperatorAuthResult =
  | { ok: true; operator: Operator }
  | { ok: false; response: NextResponse };

export async function requireOperator(
  requiredPermission?: string,
  extraHeaders?: Record<string, string>,
): Promise<OperatorAuthResult> {
  const operator = await getCurrentOperator();
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
