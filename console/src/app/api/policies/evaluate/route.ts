import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";

export async function POST(req: NextRequest) {
  const body = await req.json();
  const payload = {
    tenant_id: body.tenantId || "default",
    policy_id: body.policyId || "",
    payload: body.payload || {},
    label_filter: body.labelFilter || {},
  };

  const res = await proxyRequest("themis", "/policies/evaluate", z.any(), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

  if (!res.ok) return res;

  const data = await res.json();
  return NextResponse.json({
    AllPassed: data.all_passed ?? data.AllPassed ?? false,
    TotalEvaluated: data.total_evaluated ?? data.TotalEvaluated ?? 0,
    Results: (data.results ?? data.Results ?? []).map((r: Record<string, unknown>) => ({
      PolicyID: ((r.policy_id ?? r.PolicyID) as string) ?? "",
      PolicyName: ((r.policy_name ?? r.PolicyName) as string) ?? "",
      Passed: Boolean(r.passed ?? r.Passed ?? false),
      Error: (r.error ?? r.Error) as string | undefined,
      Expression: ((r.expression ?? r.Expression) as string) ?? "",
    })),
  });
}
