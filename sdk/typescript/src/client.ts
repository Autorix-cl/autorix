export interface RetryConfig {
  maxRetries?: number;
  initialDelayMs?: number;
  maxDelayMs?: number;
  backoffFactor?: number;
}

export interface AutorixConfig {
  baseUrl?: string;
  egoUrl?: string;
  nexusUrl?: string;
  themisUrl?: string;
  janusUrl?: string;
  vulcanUrl?: string;
  argusUrl?: string;
  apiKey?: string;
  enableCache?: boolean;
  cacheTtlMs?: number;
  retryConfig?: RetryConfig;
}

export interface UserSession {
  id: string;
  active: boolean;
  identity: {
    id: string;
    traits: {
      email?: string;
      name?: string | { first?: string; last?: string };
      department?: string;
      [key: string]: unknown;
    };
    state: string;
  };
  expiresAt: string;
  authenticatedAt?: string;
}

export interface CheckPermissionRequest {
  namespace: string;
  object: string;
  relation: string;
  subject: string;
  subjectNamespace?: string;
  context?: Record<string, unknown>;
  explain?: boolean;
}

export interface CheckPermissionResponse {
  allowed: boolean;
  reason?: string;
  trace?: Record<string, unknown>;
}

export interface EvaluatePolicyRequest {
  tenantId?: string;
  context: Record<string, unknown>;
}

export interface PolicyEvaluationResult {
  policyId: string;
  policyName: string;
  passed: boolean;
  expression: string;
  error?: string;
}

export interface EvaluatePolicyResponse {
  allPassed: boolean;
  results: PolicyEvaluationResult[];
  totalEvaluated: number;
}

export interface VerifyKeyResponse {
  valid: boolean;
  keyId?: string;
  name?: string;
  scopes?: string[];
  environment?: string;
  error?: string;
}

interface CacheItem<T> {
  data: T;
  expiresAt: number;
}

export class AutorixClient {
  private config: Required<Omit<AutorixConfig, "apiKey" | "retryConfig">> & {
    apiKey?: string;
    retryConfig: Required<RetryConfig>;
  };
  private cache = new Map<string, CacheItem<unknown>>();

  constructor(config: AutorixConfig = {}) {
    this.config = {
      baseUrl: config.baseUrl || "http://localhost:4455",
      egoUrl: config.egoUrl || "http://localhost:4433",
      nexusUrl: config.nexusUrl || "http://localhost:8080",
      themisUrl: config.themisUrl || "http://localhost:4488",
      janusUrl: config.janusUrl || "http://localhost:4444",
      vulcanUrl: config.vulcanUrl || "http://localhost:4466",
      argusUrl: config.argusUrl || "http://localhost:4400",
      apiKey: config.apiKey,
      enableCache: config.enableCache ?? true,
      cacheTtlMs: config.cacheTtlMs ?? 10_000,
      retryConfig: {
        maxRetries: config.retryConfig?.maxRetries ?? 3,
        initialDelayMs: config.retryConfig?.initialDelayMs ?? 50,
        maxDelayMs: config.retryConfig?.maxDelayMs ?? 2000,
        backoffFactor: config.retryConfig?.backoffFactor ?? 2.0,
      },
    };
  }

  private async fetchWithRetry(url: string, init: RequestInit): Promise<Response> {
    const { maxRetries, initialDelayMs, maxDelayMs, backoffFactor } = this.config.retryConfig;
    let lastError: unknown;

    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      if (attempt > 0) {
        const backoff = Math.min(initialDelayMs * Math.pow(backoffFactor, attempt - 1), maxDelayMs);
        const jitter = Math.random() * backoff;
        await new Promise((resolve) => setTimeout(resolve, jitter));
      }

      try {
        const headers = new Headers(init.headers);
        headers.set("User-Agent", "Autorix-TS-SDK/1.0.0");
        if (this.config.apiKey && !headers.has("Authorization")) {
          headers.set("Authorization", `Bearer ${this.config.apiKey}`);
        }

        const res = await fetch(url, { ...init, headers });

        // Retry on 429, 502, 503, 504
        if ([429, 502, 503, 504].includes(res.status) && attempt < maxRetries) {
          continue;
        }

        return res;
      } catch (err) {
        lastError = err;
        if (attempt === maxRetries) throw err;
      }
    }

    throw lastError || new Error(`Request failed after ${maxRetries} retries`);
  }

  /**
   * Retrieves active user session from Ego.
   */
  async whoami(token?: string): Promise<UserSession | null> {
    try {
      const headers: Record<string, string> = {};
      if (token) headers["Authorization"] = `Bearer ${token}`;

      const res = await this.fetchWithRetry(`${this.config.egoUrl}/sessions/whoami`, {
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
   * Evaluates a Zanzibar ReBAC check against Nexus with in-memory caching (Fail-Closed).
   */
  async check(req: CheckPermissionRequest): Promise<CheckPermissionResponse> {
    const cacheKey = `nexus:${req.namespace}:${req.object}#${req.relation}@${req.subjectNamespace || "user"}:${req.subject}`;

    if (this.config.enableCache) {
      const cached = this.cache.get(cacheKey) as CacheItem<CheckPermissionResponse> | undefined;
      if (cached && Date.now() < cached.expiresAt) {
        return cached.data;
      }
    }

    try {
      const baseUrl = this.config.nexusUrl.replace(/\/$/, "");
      const res = await this.fetchWithRetry(`${baseUrl}/check`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          namespace: req.namespace,
          object: req.object,
          relation: req.relation,
          subject_id: req.subject,
          subject_namespace: req.subjectNamespace || "user",
          request_context: req.context,
          explain: req.explain,
        }),
      });

      if (!res.ok) {
        return { allowed: false, reason: `Nexus returned status ${res.status}` };
      }

      const data = (await res.json()) as CheckPermissionResponse;

      if (this.config.enableCache) {
        this.cache.set(cacheKey, {
          data,
          expiresAt: Date.now() + this.config.cacheTtlMs,
        });
      }

      return data;
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Connection failed";
      return { allowed: false, reason: `Nexus check error: ${message}` };
    }
  }

  /**
   * Evaluates multiple ReBAC checks in parallel.
   */
  async checkBatch(requests: CheckPermissionRequest[]): Promise<CheckPermissionResponse[]> {
    return Promise.all(requests.map((req) => this.check(req)));
  }

  /**
   * Evaluates Google CEL ABAC policies against Themis.
   */
  async evaluatePolicy(req: EvaluatePolicyRequest): Promise<EvaluatePolicyResponse> {
    try {
      const baseUrl = this.config.themisUrl.replace(/\/$/, "");
      const res = await this.fetchWithRetry(`${baseUrl}/v1/policies/evaluate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          tenant_id: req.tenantId || "default",
          context: req.context,
        }),
      });

      if (!res.ok) {
        return { allPassed: false, results: [], totalEvaluated: 0 };
      }

      const json = await res.json();
      return {
        allPassed: json.AllPassed ?? json.all_passed ?? false,
        results: (json.Results ?? json.results ?? []).map((r: any) => ({
          policyId: r.PolicyID ?? r.policy_id,
          policyName: r.PolicyName ?? r.policy_name,
          passed: r.Passed ?? r.passed,
          expression: r.Expression ?? r.expression,
          error: r.Error ?? r.error,
        })),
        totalEvaluated: json.TotalEvaluated ?? json.total_evaluated ?? 0,
      };
    } catch {
      return { allPassed: false, results: [], totalEvaluated: 0 };
    }
  }

  /**
   * Verifies an API Key or Macaroon against Vulcan.
   */
  async verifyApiKey(token: string, context?: Record<string, unknown>): Promise<VerifyKeyResponse> {
    try {
      const baseUrl = this.config.vulcanUrl.replace(/\/$/, "");
      const res = await this.fetchWithRetry(`${baseUrl}/keys/verify`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token, context }),
      });

      if (!res.ok) {
        return { valid: false, error: `Vulcan returned status ${res.status}` };
      }

      return (await res.json()) as VerifyKeyResponse;
    } catch (err) {
      return { valid: false, error: err instanceof Error ? err.message : "Verification error" };
    }
  }
}
