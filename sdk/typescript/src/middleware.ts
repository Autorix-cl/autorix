import { createHmac, timingSafeEqual } from "crypto";

export const IDENTITY_SIGNATURE_HEADER = "x-autorix-identity-signature";

export interface AutorixAuthHeaderUser {
  id: string;
  email?: string;
  roles: string[];
}

export interface ExtractUserOptions {
  /**
   * Shared HMAC-SHA256 secret configured on Aegis's header mutator.
   * Defaults to process.env.AUTORIX_IDENTITY_SIGNING_SECRET || process.env.AEGIS_IDENTITY_SIGNING_SECRET.
   */
  signingSecret?: string;
  /**
   * Fail closed when no valid X-Autorix-Identity-Signature is present, even if signingSecret is empty.
   */
  requireSignature?: boolean;
}

function firstHeaderValue(val: string | string[] | undefined): string {
  if (typeof val === "string") return val;
  if (Array.isArray(val) && val.length > 0) return val[0] ?? "";
  return "";
}

function getHeaderCaseInsensitive(
  headers: Record<string, string | string[] | undefined>,
  name: string,
): string {
  const lower = name.toLowerCase();
  if (lower in headers) {
    return firstHeaderValue(headers[lower]);
  }
  for (const key of Object.keys(headers)) {
    if (key.toLowerCase() === lower) {
      return firstHeaderValue(headers[key]);
    }
  }
  return "";
}

/**
 * Computes the canonical v1 HMAC-SHA256 signature for Aegis-injected identity headers.
 */
export function signIdentityHeaders(
  userId: string,
  email: string | undefined,
  roles: string | undefined,
  scopes: string | undefined,
  secret: string,
): string {
  const canonical = `v1\n${userId ?? ""}\n${email ?? ""}\n${roles ?? ""}\n${scopes ?? ""}`;
  const digest = createHmac("sha256", secret).update(canonical, "utf8").digest("hex");
  return `v1=${digest}`;
}

/**
 * Verifies X-Autorix-Identity-Signature against X-User-* headers using constant-time comparison.
 */
export function verifyIdentityHeaders(
  headers: Record<string, string | string[] | undefined>,
  secret: string,
): boolean {
  const trimmedSecret = secret?.trim();
  if (!trimmedSecret) return false;

  const signature = getHeaderCaseInsensitive(headers, IDENTITY_SIGNATURE_HEADER).trim();
  if (!signature) return false;

  const expected = signIdentityHeaders(
    getHeaderCaseInsensitive(headers, "x-user-id"),
    getHeaderCaseInsensitive(headers, "x-user-email"),
    getHeaderCaseInsensitive(headers, "x-user-roles"),
    getHeaderCaseInsensitive(headers, "x-user-scopes"),
    trimmedSecret,
  );

  const sigBuf = Buffer.from(signature, "utf8");
  const expBuf = Buffer.from(expected, "utf8");
  if (sigBuf.length !== expBuf.length) return false;
  return timingSafeEqual(sigBuf, expBuf);
}

/**
 * Extracts identity headers injected by Autorix Aegis in Node.js/Express.
 * When signingSecret is configured (via options or AUTORIX_IDENTITY_SIGNING_SECRET /
 * AEGIS_IDENTITY_SIGNING_SECRET), verifies X-Autorix-Identity-Signature before trusting headers.
 */
export function extractUserFromHeaders(
  headers: Record<string, string | string[] | undefined>,
  options?: ExtractUserOptions,
): AutorixAuthHeaderUser | null {
  const userId = getHeaderCaseInsensitive(headers, "x-user-id");
  if (!userId) {
    return null;
  }

  const secret =
    options?.signingSecret !== undefined
      ? options.signingSecret
      : process.env.AUTORIX_IDENTITY_SIGNING_SECRET || process.env.AEGIS_IDENTITY_SIGNING_SECRET || "";

  if (options?.requireSignature || secret.trim() !== "") {
    if (!verifyIdentityHeaders(headers, secret)) {
      return null;
    }
  }

  const rawRoles = getHeaderCaseInsensitive(headers, "x-user-roles");
  const roles = rawRoles
    ? rawRoles
        .split(",")
        .map((r) => r.trim())
        .filter(Boolean)
    : [];

  const email = getHeaderCaseInsensitive(headers, "x-user-email") || undefined;

  return {
    id: userId,
    email,
    roles,
  };
}
