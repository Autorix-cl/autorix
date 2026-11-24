import assert from "node:assert/strict";
import test from "node:test";
import { AutorixApiError, AutorixClient } from "../dist/client.js";

function json(body, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "content-type": "application/json" } });
}

test("uses the public Themis evaluation route and maps the existing context field to payload", async () => {
  let request;
  const client = new AutorixClient({ themisUrl: "https://themis.example", fetch: async (url, init) => {
    request = { url, init };
    return json({ all_passed: true, results: [], total_evaluated: 0 });
  } });
  const response = await client.evaluatePolicy({ tenantId: "tenant-a", context: { amount: 42 } });
  assert.equal(response.allPassed, true);
  assert.equal(request.url, "https://themis.example/policies/evaluate");
  assert.deepEqual(JSON.parse(request.init.body), { tenant_id: "tenant-a", payload: { amount: 42 } });
});

test("does not retry a non-idempotent credential creation request", async () => {
  let calls = 0;
  const client = new AutorixClient({ vulcanUrl: "https://vulcan.example", retryConfig: { maxRetries: 3 }, fetch: async () => { calls++; return json({ error: "unavailable" }, 503); } });
  await assert.rejects(() => client.createApiKey({ name: "ci", ownerId: "owner" }), AutorixApiError);
  assert.equal(calls, 1);
});

test("retries safe reads and preserves a typed HTTP error", async () => {
  let calls = 0;
  const client = new AutorixClient({ janusUrl: "https://janus.example", retryConfig: { maxRetries: 1, initialDelayMs: 0, maxDelayMs: 0 }, fetch: async () => {
    calls++;
    return calls === 1 ? json({ error: "temporary" }, 503) : json({ keys: [] });
  } });
  assert.deepEqual(await client.getJwks(), { keys: [] });
  assert.equal(calls, 2);
});

test("permission failures remain fail-closed", async () => {
  const client = new AutorixClient({ nexusUrl: "https://nexus.example", fetch: async () => json({ error: "unavailable" }, 503) });
  const decision = await client.check({ namespace: "document", object: "1", relation: "view", subject: "user-1" });
  assert.equal(decision.allowed, false);
});

test("preserves base URL subpaths when joining paths", async () => {
  const seenUrls = [];
  const client = new AutorixClient({
    nexusUrl: "https://gateway.example/api/v1",
    janusUrl: "https://auth.example/oauth/",
    fetch: async (url) => {
      seenUrls.push(String(url));
      return json({ allowed: true });
    },
  });
  await client.check({ namespace: "doc", object: "1", relation: "view", subject: "alice" });
  assert.equal(seenUrls[0], "https://gateway.example/api/v1/check");
  const authUrl = client.createAuthorizationUrl({ clientId: "spa", redirectUri: "https://app.test/cb", state: "s", codeChallenge: "c" });
  assert.ok(authUrl.startsWith("https://auth.example/oauth/oauth2/auth?"), `unexpected authUrl: ${authUrl}`);
});

test("check cache bypasses context, snapToken, explain, isolates subjectRelation, and bounds size", async () => {
  let calls = 0;
  const client = new AutorixClient({
    nexusUrl: "https://nexus.example",
    enableCache: true,
    maxCacheSize: 2,
    fetch: async () => {
      calls++;
      return json({ allowed: true, reason: "ok", trace: { call: calls } });
    },
  });
  await client.check({ namespace: "doc", object: "1", relation: "view", subject: "u1" });
  await client.check({ namespace: "doc", object: "1", relation: "view", subject: "u1" }); // hit
  assert.equal(calls, 1);
  await client.check({ namespace: "doc", object: "1", relation: "view", subject: "u1", context: { ip: "1.2.3.4" } });
  await client.check({ namespace: "doc", object: "1", relation: "view", subject: "u1", snapToken: "zk1" });
  await client.check({ namespace: "doc", object: "1", relation: "view", subject: "u1", explain: true });
  await client.check({ namespace: "doc", object: "1", relation: "view", subject: "u1", subjectRelation: "member" });
  assert.equal(calls, 5);
  await client.check({ namespace: "doc", object: "2", relation: "view", subject: "u2" }); // triggers eviction for maxCacheSize=2
  assert.equal(calls, 6);
});

test("OAuth token endpoint does not leak API key Authorization header", async () => {
  let authHeader = null;
  const client = new AutorixClient({
    janusUrl: "https://janus.example",
    apiKey: "av_live_secret",
    fetch: async (_url, init) => {
      authHeader = new Headers(init?.headers).get("authorization");
      return json({ access_token: "at", token_type: "Bearer", expires_in: 60 });
    },
  });
  await client.exchangeAuthorizationCode({ code: "c", redirectUri: "https://app.test/cb", clientId: "spa", codeVerifier: "v" });
  assert.equal(authHeader, null);
});
