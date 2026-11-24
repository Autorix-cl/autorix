/**
 * Shared internal service authentication headers for Console BFF -> engine calls.
 * Sends both canonical X-Autorix-Internal-Token and legacy X-Autorix-Console-Token
 * whenever AUTORIX_INTERNAL_AUTH_SECRET or AUTORIX_CONSOLE_INTERNAL_TOKEN is set.
 */
export function getInternalAuthToken(): string | undefined {
  const token = process.env.AUTORIX_INTERNAL_AUTH_SECRET || process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN;
  return token?.trim() || undefined;
}

export function buildInternalAuthHeaders(extra?: Record<string, string>): Record<string, string> {
  const headers: Record<string, string> = {
    "x-autorix-caller": "console",
    ...extra,
  };
  const token = getInternalAuthToken();
  if (token) {
    headers["x-autorix-internal-token"] = token;
    headers["x-autorix-console-token"] = token;
  }
  return headers;
}
