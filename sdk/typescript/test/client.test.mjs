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
