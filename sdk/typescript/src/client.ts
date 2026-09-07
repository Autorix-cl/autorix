/** Public runtime TypeScript client for Autorix services. */
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
  /** Server-side credential. Do not configure this in browser applications. */
  apiKey?: string;
  enableCache?: boolean;
  cacheTtlMs?: number;
  timeoutMs?: number;
  retryConfig?: RetryConfig;
  /** Injectable transport for tests or non-browser runtimes. */
  fetch?: typeof globalThis.fetch;
}

export class AutorixApiError extends Error {
  readonly status: number;
  readonly code?: string;
  readonly body?: unknown;
  constructor(status: number, message: string, body?: unknown) {
    super(message);
    this.name = "AutorixApiError";
    this.status = status;
    this.body = body;
    this.code = typeof body === "object" && body !== null && "code" in body && typeof body.code === "string" ? body.code : undefined;
  }
}

export class AutorixTimeoutError extends Error {
  constructor(readonly timeoutMs: number) {
    super(`Autorix request timed out after ${timeoutMs}ms`);
    this.name = "AutorixTimeoutError";
  }
}

export interface UserSession { id: string; active: boolean; identity: Identity; expiresAt: string; authenticatedAt?: string; }
export interface Identity { id: string; traits: Record<string, unknown> & { email?: string; name?: string | { first?: string; last?: string }; department?: string }; state: string; }
export interface CheckPermissionRequest { namespace: string; object: string; relation: string; subject: string; subjectNamespace?: string; subjectRelation?: string; context?: Record<string, unknown>; explain?: boolean; snapToken?: string; }
export interface CheckPermissionResponse { allowed: boolean; reason?: string; trace?: Record<string, unknown>; snapToken?: string; }
export interface EvaluatePolicyRequest { tenantId?: string; policyId?: string; /** Alias retained for 1.0.0 compatibility. */ context: Record<string, unknown>; labelFilter?: Record<string, string>; }
export interface PolicyEvaluationResult { policyId: string; policyName: string; passed: boolean; expression: string; error?: string; }
export interface EvaluatePolicyResponse { allPassed: boolean; results: PolicyEvaluationResult[]; totalEvaluated: number; }
export interface VerifyKeyResponse { valid: boolean; keyId?: string; name?: string; scopes?: string[]; environment?: string; error?: string; apiKey?: ApiKey; }
export interface RelationTuple { namespace: string; object: string; relation: string; subjectNamespace?: string; subjectId: string; subjectRelation?: string; caveatName?: string; caveatContext?: Record<string, unknown>; commitTime?: string; }
export interface Page<T> { data: T[]; nextCursor?: string; hasMore: boolean; }
export interface Policy { id: string; tenantId: string; name: string; description: string; expression: string; priority: number; enabled: boolean; labels: Record<string, string>; createdAt: string; updatedAt: string; }
export type PolicyInput = Omit<Policy, "id" | "createdAt" | "updatedAt">;
export interface PolicyVersion extends Policy { policyId: string; version: number; }
export interface PolicyFixture { id: string; policyId: string; tenantId: string; name: string; description: string; payload: Record<string, unknown>; expectedResult: boolean; createdAt: string; updatedAt: string; }
export interface ValidationResult { valid: boolean; variables: string[]; errors?: Array<{ line: number; column: number; message: string }>; }
export interface DryRunResult { passed: boolean; error?: string; }
export interface OpenIdConfiguration { issuer: string; authorization_endpoint: string; token_endpoint: string; introspection_endpoint: string; revocation_endpoint: string; jwks_uri: string; code_challenge_methods_supported: string[]; [key: string]: unknown; }
export interface Jwk { kty: string; kid?: string; use?: string; alg?: string; n?: string; e?: string; [key: string]: unknown; }
export interface Jwks { keys: Jwk[]; }
export interface TokenResponse { access_token: string; token_type: string; expires_in?: number; refresh_token?: string; id_token?: string; scope?: string; }
export interface TokenIntrospectionResponse { active: boolean; scope?: string; client_id?: string; sub?: string; exp?: number; iat?: number; iss?: string; aud?: string[]; [key: string]: unknown; }
export interface Macaroon { location: string; key_id: string; caveats: Array<{ predicate: string }>; signature: string; }
export interface ApiKey { id: string; key_prefix: string; key_hint: string; name: string; description: string; owner_id: string; scopes: string[]; expires_at?: string; state: string; created_at: string; updated_at: string; [key: string]: unknown; }
export interface CreateApiKeyRequest { name: string; ownerId: string; description?: string; scopes?: string[]; expiresAt?: string; isLive?: boolean; }
export interface CreateApiKeyResponse { api_key: ApiKey; raw_token: string; macaroon: Macaroon; }

interface CacheItem<T> { data: T; expiresAt: number; }
type HttpMethod = "GET" | "HEAD" | "OPTIONS" | "POST" | "PUT" | "PATCH" | "DELETE";

export class AutorixClient {
  private readonly config: Required<Omit<AutorixConfig, "apiKey" | "retryConfig" | "fetch">> & { apiKey?: string; retryConfig: Required<RetryConfig>; fetch: typeof globalThis.fetch };
  private readonly cache = new Map<string, CacheItem<unknown>>();

  constructor(config: AutorixConfig = {}) {
    this.config = {
      baseUrl: config.baseUrl ?? "http://localhost:4455", egoUrl: config.egoUrl ?? "http://localhost:4433", nexusUrl: config.nexusUrl ?? "http://localhost:8080",
      themisUrl: config.themisUrl ?? "http://localhost:4488", janusUrl: config.janusUrl ?? "http://localhost:4444", vulcanUrl: config.vulcanUrl ?? "http://localhost:4466", argusUrl: config.argusUrl ?? "http://localhost:4400",
      apiKey: config.apiKey, enableCache: config.enableCache ?? true, cacheTtlMs: config.cacheTtlMs ?? 10_000, timeoutMs: config.timeoutMs ?? 10_000,
      retryConfig: { maxRetries: config.retryConfig?.maxRetries ?? 3, initialDelayMs: config.retryConfig?.initialDelayMs ?? 50, maxDelayMs: config.retryConfig?.maxDelayMs ?? 2_000, backoffFactor: config.retryConfig?.backoffFactor ?? 2 },
      fetch: config.fetch ?? globalThis.fetch,
    };
    if (!this.config.fetch) throw new Error("A Fetch implementation is required. Provide config.fetch for this runtime.");
  }

  private url(base: string, path: string, query?: Record<string, string | number | boolean | undefined>): string {
    const result = new URL(path, base.endsWith("/") ? base : `${base}/`);
    for (const [key, value] of Object.entries(query ?? {})) if (value !== undefined) result.searchParams.set(key, String(value));
    return result.toString();
  }

  private async request<T>(base: string, path: string, init: RequestInit & { retry?: boolean } = {}, query?: Record<string, string | number | boolean | undefined>): Promise<T> {
    const method = (init.method ?? "GET").toUpperCase() as HttpMethod;
    // Retrying non-idempotent writes can duplicate credentials, policies, or relationships.
    const retry = init.retry ?? ["GET", "HEAD", "OPTIONS"].includes(method);
    const headers = new Headers(init.headers);
    headers.set("X-Autorix-SDK", "typescript/1.0.0");
    if (this.config.apiKey && !headers.has("Authorization")) headers.set("Authorization", `Bearer ${this.config.apiKey}`);
    const target = this.url(base, path, query);
    let lastError: unknown;
    for (let attempt = 0; attempt <= this.config.retryConfig.maxRetries; attempt++) {
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), this.config.timeoutMs);
      const forwardAbort = () => controller.abort();
      init.signal?.addEventListener("abort", forwardAbort, { once: true });
      try {
        const response = await this.config.fetch(target, { ...init, method, headers, signal: controller.signal });
        if (response.ok) {
          if (response.status === 204) return undefined as T;
          return (await response.json()) as T;
        }
        if (retry && [429, 500, 502, 503, 504].includes(response.status) && attempt < this.config.retryConfig.maxRetries) {
          await response.body?.cancel();
        } else {
          const body = await response.json().catch(() => undefined);
          const message = typeof body === "object" && body !== null && "error" in body && typeof body.error === "string" ? body.error : `Autorix returned status ${response.status}`;
          throw new AutorixApiError(response.status, message, body);
        }
      } catch (error) {
        lastError = controller.signal.aborted && !init.signal?.aborted ? new AutorixTimeoutError(this.config.timeoutMs) : error;
        if (!retry || attempt === this.config.retryConfig.maxRetries || error instanceof AutorixApiError) throw lastError;
      } finally {
        clearTimeout(timeout);
        init.signal?.removeEventListener("abort", forwardAbort);
      }
      const maxDelay = Math.min(this.config.retryConfig.initialDelayMs * Math.pow(this.config.retryConfig.backoffFactor, attempt), this.config.retryConfig.maxDelayMs);
      await new Promise<void>((resolve) => setTimeout(resolve, Math.random() * maxDelay));
    }
    throw lastError ?? new Error("Autorix request failed");
  }

  /** Retrieves the active Ego session. Invalid sessions intentionally return null. */
  async whoami(token?: string, signal?: AbortSignal): Promise<UserSession | null> {
    try { return await this.request<UserSession>(this.config.egoUrl, "/sessions/whoami", { headers: token ? { Authorization: `Bearer ${token}` } : undefined, credentials: "include", signal }); }
    catch (error) { if (error instanceof AutorixApiError && [401, 403, 404].includes(error.status)) return null; throw error; }
  }
  async logout(signal?: AbortSignal): Promise<void> { return this.request<void>(this.config.egoUrl, "/self-service/logout", { method: "POST", credentials: "include", signal }); }

  /** Fails closed: transport and API failures become denied decisions. */
  async check(req: CheckPermissionRequest, signal?: AbortSignal): Promise<CheckPermissionResponse> {
    const cacheKey = `nexus:${req.namespace}:${req.object}#${req.relation}@${req.subjectNamespace ?? "user"}:${req.subject}`;
    const cached = this.cache.get(cacheKey) as CacheItem<CheckPermissionResponse> | undefined;
    if (this.config.enableCache && cached && Date.now() < cached.expiresAt) return cached.data;
    try {
      const response = await this.request<CheckPermissionResponse>(this.config.nexusUrl, "/check", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ namespace: req.namespace, object: req.object, relation: req.relation, subject_id: req.subject, subject_namespace: req.subjectNamespace ?? "user", subject_relation: req.subjectRelation, request_context: req.context, explain: req.explain, snap_token: req.snapToken }), signal });
      if (this.config.enableCache) this.cache.set(cacheKey, { data: response, expiresAt: Date.now() + this.config.cacheTtlMs });
      return response;
    } catch (error) { return { allowed: false, reason: error instanceof Error ? error.message : "Nexus check failed" }; }
  }
  async checkBatch(requests: CheckPermissionRequest[], signal?: AbortSignal): Promise<CheckPermissionResponse[]> { return Promise.all(requests.map((request) => this.check(request, signal))); }
  async listTuples(namespace?: string, page?: { limit?: number; cursor?: string }, signal?: AbortSignal): Promise<Page<RelationTuple>> { return this.request<Page<RelationTuple>>(this.config.nexusUrl, "/tuples", { signal }, { namespace, limit: page?.limit, cursor: page?.cursor }); }
  async writeTuples(tuples: RelationTuple[], signal?: AbortSignal): Promise<void> { return this.request<void>(this.config.nexusUrl, "/tuples", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ tuples: tuples.map(toWireTuple) }), signal }); }
  async deleteTuples(tuples: RelationTuple[], signal?: AbortSignal): Promise<void> { return this.request<void>(this.config.nexusUrl, "/tuples", { method: "DELETE", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ tuples: tuples.map(toWireTuple) }), signal }); }
  async expand(namespace: string, object: string, relation: string, signal?: AbortSignal): Promise<Record<string, unknown>> { return this.request(this.config.nexusUrl, "/expand", { signal }, { namespace, object, relation }); }
  async lookupSubjects(namespace: string, object: string, relation: string, signal?: AbortSignal): Promise<RelationTuple[]> { const result = await this.request<{ subjects: RelationTuple[] }>(this.config.nexusUrl, "/lookup/subjects", { signal }, { namespace, object, relation }); return result.subjects; }
  async lookupResources(namespace: string, relation: string, subject: string, subjectNamespace = "user", signal?: AbortSignal): Promise<string[]> { const result = await this.request<{ resources: string[] }>(this.config.nexusUrl, "/lookup/resources", { signal }, { namespace, relation, subject_id: subject, subject_namespace: subjectNamespace }); return result.resources; }

  async evaluatePolicy(req: EvaluatePolicyRequest, signal?: AbortSignal): Promise<EvaluatePolicyResponse> {
    try {
      const response = await this.request<Record<string, unknown>>(this.config.themisUrl, "/policies/evaluate", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ tenant_id: req.tenantId ?? "default", policy_id: req.policyId, payload: req.context, label_filter: req.labelFilter }), signal });
      return {
        allPassed: Boolean(response.allPassed ?? response.all_passed),
        results: Array.isArray(response.results) ? response.results as PolicyEvaluationResult[] : [],
        totalEvaluated: Number(response.totalEvaluated ?? response.total_evaluated ?? 0),
      };
    }
    catch { return { allPassed: false, results: [], totalEvaluated: 0 }; }
  }
  async listPolicies(options: { tenantId?: string; enabledOnly?: boolean; limit?: number; cursor?: string } = {}, signal?: AbortSignal): Promise<Page<Policy>> { return this.request<Page<Policy>>(this.config.themisUrl, "/policies", { signal }, { tenant_id: options.tenantId, enabled_only: options.enabledOnly, limit: options.limit, cursor: options.cursor }); }
  async getPolicy(id: string, tenantId?: string, signal?: AbortSignal): Promise<Policy> { return this.request<Policy>(this.config.themisUrl, `/policies/${encodeURIComponent(id)}`, { signal }, { tenant_id: tenantId }); }
  async createPolicy(policy: PolicyInput, signal?: AbortSignal): Promise<Policy> { return this.request<Policy>(this.config.themisUrl, "/policies", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(toSnakePolicy(policy)), signal }); }
  async updatePolicy(id: string, policy: PolicyInput, signal?: AbortSignal): Promise<Policy> { return this.request<Policy>(this.config.themisUrl, `/policies/${encodeURIComponent(id)}`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(toSnakePolicy(policy)), signal }); }
  async deletePolicy(id: string, tenantId?: string, signal?: AbortSignal): Promise<{ status: string }> { return this.request(this.config.themisUrl, `/policies/${encodeURIComponent(id)}`, { method: "DELETE", signal }, { tenant_id: tenantId }); }
  async listPolicyVersions(id: string, tenantId?: string, signal?: AbortSignal): Promise<PolicyVersion[]> { return this.request(this.config.themisUrl, `/policies/${encodeURIComponent(id)}/versions`, { signal }, { tenant_id: tenantId }); }
  async validatePolicy(expression: string, signal?: AbortSignal): Promise<ValidationResult> { return this.request(this.config.themisUrl, "/policies/validate", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ expression }), signal }); }
  async dryRunPolicy(expression: string, payload: Record<string, unknown>, signal?: AbortSignal): Promise<DryRunResult> { return this.request(this.config.themisUrl, "/policies/dry-run", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ expression, payload }), signal }); }

  async getOpenIdConfiguration(signal?: AbortSignal): Promise<OpenIdConfiguration> { return this.request(this.config.janusUrl, "/.well-known/openid-configuration", { signal }); }
  async getJwks(signal?: AbortSignal): Promise<Jwks> { return this.request(this.config.janusUrl, "/.well-known/jwks.json", { signal }); }
  createAuthorizationUrl(options: { clientId: string; redirectUri: string; state: string; codeChallenge: string; scope?: string; nonce?: string; }): string { return this.url(this.config.janusUrl, "/oauth2/auth", { client_id: options.clientId, redirect_uri: options.redirectUri, response_type: "code", state: options.state, code_challenge: options.codeChallenge, code_challenge_method: "S256", scope: options.scope ?? "openid", nonce: options.nonce }); }
  async exchangeAuthorizationCode(options: { code: string; redirectUri: string; clientId: string; codeVerifier: string; clientSecret?: string }, signal?: AbortSignal): Promise<TokenResponse> { return this.token({ grant_type: "authorization_code", code: options.code, redirect_uri: options.redirectUri, client_id: options.clientId, code_verifier: options.codeVerifier, client_secret: options.clientSecret }, signal); }
  async refreshToken(refreshToken: string, clientId: string, clientSecret?: string, signal?: AbortSignal): Promise<TokenResponse> { return this.token({ grant_type: "refresh_token", refresh_token: refreshToken, client_id: clientId, client_secret: clientSecret }, signal); }
  private async token(params: Record<string, string | undefined>, signal?: AbortSignal): Promise<TokenResponse> { const body = new URLSearchParams(); for (const [key, value] of Object.entries(params)) if (value !== undefined) body.set(key, value); return this.request(this.config.janusUrl, "/oauth2/token", { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body, signal }); }
  async introspectToken(token: string, signal?: AbortSignal): Promise<TokenIntrospectionResponse> { return this.request(this.config.janusUrl, "/oauth2/introspect", { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body: new URLSearchParams({ token }), signal }); }
  async revokeToken(token: string, signal?: AbortSignal): Promise<void> { return this.request<void>(this.config.janusUrl, "/oauth2/revoke", { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body: new URLSearchParams({ token }), signal }); }

  /** Fails closed for capability verification. */
  async verifyApiKey(macaroon: Macaroon | string, context?: Record<string, unknown>, signal?: AbortSignal): Promise<VerifyKeyResponse> {
    // `string` is retained for source compatibility with 1.0.0. Vulcan's
    // public runtime endpoint validates structured macaroons; strings fail closed.
    const body = typeof macaroon === "string" ? { token: macaroon, context } : { macaroon, context };
    try { return await this.request(this.config.vulcanUrl, "/keys/verify", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body), signal }); }
    catch (error) { return { valid: false, error: error instanceof Error ? error.message : "Verification failed" }; }
  }
  async createApiKey(request: CreateApiKeyRequest, signal?: AbortSignal): Promise<CreateApiKeyResponse> { return this.request(this.config.vulcanUrl, "/keys", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name: request.name, owner_id: request.ownerId, description: request.description, scopes: request.scopes ?? [], expires_at: request.expiresAt, is_live: request.isLive ?? false }), signal }); }
  async listApiKeys(page?: { limit?: number; cursor?: string }, signal?: AbortSignal): Promise<Page<ApiKey>> { return this.request(this.config.vulcanUrl, "/keys", { signal }, { limit: page?.limit, cursor: page?.cursor }); }
  async attenuateMacaroon(macaroon: Macaroon, caveat: string, signal?: AbortSignal): Promise<Macaroon> { return this.request(this.config.vulcanUrl, "/keys/attenuate", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ macaroon, caveat }), signal }); }
  async revokeApiKey(id: string, signal?: AbortSignal): Promise<{ status: string }> { return this.request(this.config.vulcanUrl, `/keys/${encodeURIComponent(id)}`, { method: "DELETE", signal }); }
}

function toWireTuple(tuple: RelationTuple): Record<string, unknown> { return { namespace: tuple.namespace, object: tuple.object, relation: tuple.relation, subject_namespace: tuple.subjectNamespace, subject_id: tuple.subjectId, subject_relation: tuple.subjectRelation, caveat_name: tuple.caveatName, caveat_context: tuple.caveatContext }; }
function toSnakePolicy(policy: PolicyInput): Record<string, unknown> { return { tenant_id: policy.tenantId, name: policy.name, description: policy.description, expression: policy.expression, priority: policy.priority, enabled: policy.enabled, labels: policy.labels }; }
