export interface AutorixAuthHeaderUser {
  id: string;
  email?: string;
  roles: string[];
}

/**
 * Extracts identity headers injected by Autorix Aegis in Node.js/Express
 */
export function extractUserFromHeaders(
  headers: Record<string, string | string[] | undefined>
): AutorixAuthHeaderUser | null {
  const userId = headers["x-user-id"];
  if (!userId || typeof userId !== "string") {
    return null;
  }

  const rawRoles = headers["x-user-roles"];
  let roles: string[] = [];
  if (typeof rawRoles === "string") {
    roles = rawRoles.split(",");
  } else if (Array.isArray(rawRoles)) {
    roles = rawRoles;
  }

  const email = typeof headers["x-user-email"] === "string" ? headers["x-user-email"] : undefined;

  return {
    id: userId,
    email,
    roles,
  };
}
