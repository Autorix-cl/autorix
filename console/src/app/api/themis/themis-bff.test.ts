import { describe, it, expect, vi, beforeEach } from "vitest";
import { NextRequest } from "next/server";
import { GET as listPolicies, POST as createPolicy } from "./policies/route";
import { GET as getPolicy, PUT as updatePolicy, DELETE as deletePolicy } from "./policies/[id]/route";
import { PATCH as togglePolicy } from "./policies/[id]/toggle/route";
import { POST as evaluatePolicy } from "./eval/route";
import { POST as validatePolicy } from "./validate/route";
import { GET as listVersions } from "./policies/[id]/versions/route";
import { POST as rollbackPolicy } from "./policies/[id]/rollback/[version]/route";
import { GET as listFixtures, POST as createFixture } from "./policies/[id]/fixtures/route";
import { DELETE as deleteFixture } from "./policies/[id]/fixtures/[fixture_id]/route";
import { POST as runTestSuite } from "./policies/[id]/test-suite/route";

describe("Themis BFF Proxy Routes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn();
  });

  it("lists policies via proxy", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify([{ id: "p1", name: "Policy 1", expression: "true", priority: 1, enabled: true }]), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    const req = new NextRequest("http://localhost/api/themis/policies?tenant_id=default");
    const res = await listPolicies(req);
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json).toHaveLength(1);
    expect(json[0].name).toBe("Policy 1");
  });

  it("creates a policy via proxy", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ id: "p1", name: "New Policy", expression: "true" }), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      })
    );

    const req = new NextRequest("http://localhost/api/themis/policies", {
      method: "POST",
      body: JSON.stringify({ name: "New Policy", expression: "true", priority: 1 }),
    });
    const res = await createPolicy(req);
    expect(res.status).toBe(201);
    const json = await res.json();
    expect(json.name).toBe("New Policy");
  });

  it("gets, updates, and deletes a single policy", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ id: "p1", name: "Policy 1" }), { status: 200, headers: { "Content-Type": "application/json" } })
    );
    const getRes = await getPolicy(new NextRequest("http://localhost/api/themis/policies/p1"), { params: Promise.resolve({ id: "p1" }) });
    expect(getRes.status).toBe(200);

    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ id: "p1", name: "Updated Policy" }), { status: 200, headers: { "Content-Type": "application/json" } })
    );
    const putRes = await updatePolicy(
      new NextRequest("http://localhost/api/themis/policies/p1", { method: "PUT", body: JSON.stringify({ name: "Updated Policy" }) }),
      { params: Promise.resolve({ id: "p1" }) }
    );
    expect(putRes.status).toBe(200);

    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ status: "deleted" }), { status: 200, headers: { "Content-Type": "application/json" } })
    );
    const delRes = await deletePolicy(new NextRequest("http://localhost/api/themis/policies/p1"), { params: Promise.resolve({ id: "p1" }) });
    expect(delRes.status).toBe(200);
  });

  it("toggles a policy status", async () => {
    // 1st call: GET existing policy
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ id: "p1", name: "Policy 1", enabled: true }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    // 2nd call: PUT updated policy
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ id: "p1", name: "Policy 1", enabled: false }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    const req = new NextRequest("http://localhost/api/themis/policies/p1/toggle", {
      method: "PATCH",
      body: JSON.stringify({ enabled: false }),
    });
    const res = await togglePolicy(req, { params: Promise.resolve({ id: "p1" }) });
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.enabled).toBe(false);
  });

  it("evaluates dry-run expression and full tenant evaluate", async () => {
    // Dry-run expression
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ passed: true, error: "" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    const dryRunReq = new NextRequest("http://localhost/api/themis/eval", {
      method: "POST",
      body: JSON.stringify({ expression: "request.role == 'admin'", context: { role: "admin" } }),
    });
    const dryRunRes = await evaluatePolicy(dryRunReq);
    expect(dryRunRes.status).toBe(200);
    const dryRunJson = await dryRunRes.json();
    expect(dryRunJson.result).toBe("Passed");

    // Full evaluate
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          all_passed: true,
          total_evaluated: 1,
          results: [{ policy_id: "p1", policy_name: "Admin Rule", passed: true, expression: "role == 'admin'" }],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const evalReq = new NextRequest("http://localhost/api/themis/eval", {
      method: "POST",
      body: JSON.stringify({ context: { role: "admin" } }),
    });
    const evalRes = await evaluatePolicy(evalReq);
    expect(evalRes.status).toBe(200);
    const evalJson = await evalRes.json();
    expect(evalJson.result).toBe("Passed");
    expect(evalJson.matchedPolicy.name).toBe("Admin Rule");
  });

  it("validates CEL expressions", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ valid: true, variables: ["request.role"], errors: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    const req = new NextRequest("http://localhost/api/themis/validate", {
      method: "POST",
      body: JSON.stringify({ expression: "request.role == 'admin'" }),
    });
    const res = await validatePolicy(req);
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.valid).toBe(true);
  });

  it("handles versions and rollback", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify([
          {
            id: "pv_1",
            policy_id: "p1",
            version: 1,
            name: "Policy v1",
            expression: "true",
            priority: 1,
            enabled: true,
            labels: {},
            created_at: "2026-09-02T10:00:00Z",
          },
        ]),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const vRes = await listVersions(new NextRequest("http://localhost/api/themis/policies/p1/versions"), {
      params: Promise.resolve({ id: "p1" }),
    });
    expect(vRes.status).toBe(200);
    const versions = await vRes.json();
    expect(versions).toHaveLength(1);

    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ id: "p1", name: "Rolled back" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    const rbRes = await rollbackPolicy(
      new NextRequest("http://localhost/api/themis/policies/p1/rollback/1", { method: "POST" }),
      { params: Promise.resolve({ id: "p1", version: "1" }) }
    );
    expect(rbRes.status).toBe(200);
  });

  it("handles fixtures and test suites", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify([
          {
            id: "fix_1",
            policy_id: "p1",
            name: "fixture 1",
            expected_result: true,
            payload: {},
            created_at: "2026-09-02T10:00:00Z",
            updated_at: "2026-09-02T10:00:00Z",
          },
        ]),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const fRes = await listFixtures(new NextRequest("http://localhost/api/themis/policies/p1/fixtures"), {
      params: Promise.resolve({ id: "p1" }),
    });
    expect(fRes.status).toBe(200);

    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          policy_id: "p1",
          all_passed: true,
          total_tests: 1,
          passed_tests: 1,
          failed_tests: 0,
          results: [
            {
              fixture_id: "fix_1",
              fixture_name: "fixture 1",
              expected_result: true,
              actual_result: true,
              passed: true,
              error: "",
            },
          ],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const tsRes = await runTestSuite(
      new NextRequest("http://localhost/api/themis/policies/p1/test-suite", { method: "POST" }),
      { params: Promise.resolve({ id: "p1" }) }
    );
    expect(tsRes.status).toBe(200);
    const tsJson = await tsRes.json();
    expect(tsJson.all_passed).toBe(true);

    // Create fixture
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: "fix_2",
          policy_id: "p1",
          name: "fixture 2",
          expected_result: false,
          payload: {},
          created_at: "2026-09-02T10:00:00Z",
          updated_at: "2026-09-02T10:00:00Z",
        }),
        { status: 201, headers: { "Content-Type": "application/json" } }
      )
    );

    const createFixRes = await createFixture(
      new NextRequest("http://localhost/api/themis/policies/p1/fixtures", {
        method: "POST",
        body: JSON.stringify({ name: "fixture 2", expected_result: false, payload: {} }),
      }),
      { params: Promise.resolve({ id: "p1" }) }
    );
    expect(createFixRes.status).toBe(201);

    // Delete fixture
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ status: "deleted" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    const delFixRes = await deleteFixture(
      new NextRequest("http://localhost/api/themis/policies/p1/fixtures/fix_2", { method: "DELETE" }),
      { params: Promise.resolve({ id: "p1", fixture_id: "fix_2" }) }
    );
    expect(delFixRes.status).toBe(200);
  });
});
