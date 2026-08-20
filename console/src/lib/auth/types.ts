export type OperatorRole = "owner" | "admin" | "operator" | "auditor";

export interface Operator {
  id: string;
  email: string;
  name: string;
  role: OperatorRole;
  is_local: boolean;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface AuthStatus {
  bootstrapped: boolean;
  operators_count: number;
}

export interface SessionResponse {
  valid: boolean;
  operator: Operator;
  session: {
    id: string;
    operator_id: string;
    user_agent: string;
    ip_address: string;
    expires_at: string;
    last_active_at: string;
  };
}

export const ROLE_PERMISSIONS: Record<OperatorRole, string[]> = {
  owner: ["*"],
  admin: [
    "identities:*",
    "oauth2:*",
    "api-keys:*",
    "policies:*",
    "permissions:*",
    "proxy-rules:*",
    "enterprise:*",
    "fleet:read",
    "fleet:manage",
    "audit:*",
    "governance:*",
    "compliance:*",
  ],
  operator: [
    "identities:read",
    "identities:write",
    "oauth2:read",
    "api-keys:read",
    "policies:read",
    "permissions:read",
    "proxy-rules:read",
    "enterprise:read",
    "fleet:read",
    "audit:read",
    "governance:read",
    "compliance:read",
  ],
  auditor: [
    "identities:read",
    "oauth2:read",
    "api-keys:read",
    "policies:read",
    "permissions:read",
    "proxy-rules:read",
    "enterprise:read",
    "fleet:read",
    "audit:read",
    "governance:read",
    "compliance:read",
  ],
};

export function hasPermission(role: OperatorRole, requiredPermission: string): boolean {
  const perms = ROLE_PERMISSIONS[role] || [];
  if (perms.includes("*")) return true;
  if (perms.includes(requiredPermission)) return true;
  const [domain] = requiredPermission.split(":");
  return perms.includes(`${domain}:*`);
}
