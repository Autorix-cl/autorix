export interface AutorixConfig {
  baseUrl?: string;
  egoUrl?: string;
  nexusUrl?: string;
  janusUrl?: string;
  vulcanUrl?: string;
}

export interface UserSession {
  id: string;
  identity: {
    id: string;
    traits: {
      email?: string;
      name?: { first?: string; last?: string };
      [key: string]: unknown;
    };
    state: string;
  };
  expiresAt: string;
  authenticatedAt: string;
}

export interface CheckPermissionRequest {
  namespace: string;
  object: string;
  relation: string;
  subject: string;
  context?: Record<string, unknown>;
}

export interface CheckPermissionResponse {
  allowed: boolean;
  reason?: string;
}

export class AutorixClient {
  private config: AutorixConfig;

  constructor(config: AutorixConfig = {}) {
    this.config = {
      baseUrl: config.baseUrl || "http://localhost:4455",
      egoUrl: config.egoUrl || "http://localhost:4433",
      nexusUrl: config.nexusUrl || "http://localhost:8080",
      janusUrl: config.janusUrl || "http://localhost:4444",
      vulcanUrl: config.vulcanUrl || "http://localhost:4466",
    };
  }

  /**
   * Retrieves the currently active user session from Autorix Ego
   */
  async whoami(token?: string): Promise<UserSession | null> {
    try {
      const headers: Record<string, string> = {};
      if (token) {
        headers["Authorization"] = `Bearer ${token}`;
      }

      const res = await fetch(`${this.config.egoUrl}/sessions/whoami`, {
        method: "GET",
        headers,
        credentials: "include",
      });

      if (!res.ok) return null;
      return (await res.json()) as UserSession;
    } catch {
      return null;
    }
  }

  /**
   * Evaluates a Zanzibar ReBAC + CEL permission check (Fail-Closed)
   */
  async check(req: CheckPermissionRequest): Promise<CheckPermissionResponse> {
    try {
      const baseUrl = (this.config.nexusUrl || "http://localhost:8080").replace(/\/$/, "");
      const res = await fetch(`${baseUrl}/check`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      });

      if (!res.ok) {
        return { allowed: false, reason: `Nexus returned status ${res.status}` };
      }
      return (await res.json()) as CheckPermissionResponse;
    } catch (err: unknown) {
      // Fail-closed on network or runtime failure
      const message = err instanceof Error ? err.message : "Connection failed";
      return { allowed: false, reason: `Nexus check error: ${message}` };
    }
  }
}
