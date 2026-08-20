import { cookies } from "next/headers";
import { getServiceUrl } from "../api-config";
import type { Operator, SessionResponse } from "./types";

export const SESSION_COOKIE_NAME = "autorix_session";

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
