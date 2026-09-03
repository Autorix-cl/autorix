import { NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";
import { themisDb } from "@/lib/server/themis-db";
import { evalCel } from "@/lib/server/themis-evaluator";

export async function POST(req: Request) {
  try {
    const body = await req.json();
    const context = body.context || body.payload;

    if (!context) {
      return NextResponse.json({ error: "Missing context" }, { status: 400 });
    }

    // 1. If mock policies exist in themisDb (e.g. during themisDb unit tests), evaluate against them:
    const localPolicies = themisDb.getPolicies();
    if (localPolicies && localPolicies.length > 0) {
      if (body.expression) {
        const allowed = evalCel(body.expression, context);
        return NextResponse.json({
          result: allowed ? "Passed" : "Failed",
          matchedPolicy: null,
        });
      }

      const activePolicies = localPolicies.filter((p: { enabled?: boolean }) => p.enabled);
      for (const policy of activePolicies) {
        const allowed = evalCel(policy.expression, context);
        if (allowed) {
          return NextResponse.json({
            result: "Passed",
            matchedPolicy: policy,
          });
        }
      }
      return NextResponse.json({
        result: "Failed",
        matchedPolicy: null,
      });
    }

    // 2. Otherwise, proxy to real Themis Go engine:
    if (body.expression) {
      const res = await proxyRequest("themis", "/policies/dry-run", z.any(), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          expression: body.expression,
          payload: context,
        }),
      });

      if (!res.ok) return res;
      const data = await res.json();
      return NextResponse.json({
        result: data.passed ? "Passed" : "Failed",
        matchedPolicy: null,
        error: data.error || undefined,
      });
    }

    const res = await proxyRequest("themis", "/policies/evaluate", z.any(), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        tenant_id: body.tenant_id || body.tenantId || "default",
        policy_id: body.policy_id || body.policyId || "",
        payload: context,
        label_filter: body.label_filter || body.labelFilter || {},
      }),
    });

    if (!res.ok) return res;
    const data = await res.json();
    const results = data.results || [];
    const matched = results.find((r: { passed?: boolean }) => r.passed);

    return NextResponse.json({
      result: data.all_passed || Boolean(matched) ? "Passed" : "Failed",
      matchedPolicy: matched
        ? { id: matched.policy_id, name: matched.policy_name, expression: matched.expression }
        : null,
      results,
      all_passed: data.all_passed,
    });


    return res;
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Evaluation failed";
    return NextResponse.json({ error: message }, { status: 500 });
  }
}



