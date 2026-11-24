import { NextRequest, NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";
import { requireOperator } from "@/lib/auth/session";
import { safePatternMatch } from "@/lib/explorer/pattern-match";
import type { EffectiveAccessResult } from "@/lib/api/schemas/explorer";

const EXPLORER_TIMEOUT_MS = 8_000;

function buildInternalHeaders(json = false): Record<string, string> {
  const headers: Record<string, string> = { "x-autorix-caller": "console" };
  if (json) headers["Content-Type"] = "application/json";
  if (process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN) {
    headers["x-autorix-console-token"] = process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN;
  }
  return headers;
}

export async function POST(req: NextRequest) {
  const auth = await requireOperator("explorer:read");
  if (!auth.ok) return auth.response;

  try {
    const body = await req.json();
    const {
      subject = "user:alice",
      resource = "/api/v1/finance/invoices",
      action = "GET",
      context = {},
    } = body;

    const cleanSubjectId = String(subject).replace(/^user:/, "");

    // 1. Evaluate Aegis Rules (fail closed by default)
    let aegisMatched = false;
    let aegisRuleId: string | undefined;
    let aegisRuleName: string | undefined;
    let upstream: string | undefined;
    let aegisError: string | undefined;

    try {
      const aegisRes = await fetch(`${getServiceUrl("aegis")}/rules`, {
        method: "GET",
        headers: buildInternalHeaders(),
        cache: "no-store",
        signal: AbortSignal.timeout(EXPLORER_TIMEOUT_MS),
      });
      if (aegisRes.ok) {
        const rules = await aegisRes.json();
        const list = (Array.isArray(rules) ? rules : rules.rules || rules.data || []) as Record<string, unknown>[];
        const match = list.find((r) => {
          if (r.enabled === false) return false;
          if (typeof r.path === "string" && resource.startsWith(r.path)) return true;
          if (typeof r.pattern === "string" && safePatternMatch(r.pattern, resource)) return true;
          const matchObj = r.match as Record<string, unknown> | undefined;
          if (matchObj && typeof matchObj.url === "string" && safePatternMatch(matchObj.url, resource)) return true;
          return false;
        });
        if (match) {
          aegisMatched = true;
          aegisRuleId = match.id ? String(match.id) : undefined;
          aegisRuleName = String(match.name || match.id || "matched-rule");
          const upstreamObj = match.upstream as Record<string, unknown> | undefined;
          upstream = String(match.upstream_url || upstreamObj?.url || "");
        }
      } else {
        aegisError = `Aegis returned HTTP ${aegisRes.status}`;
      }
    } catch {
      aegisMatched = false;
      aegisError = "Aegis engine unreachable";
    }

    // 2. Evaluate Nexus ReBAC Relation (fail closed by default)
    const pathParts = String(resource).replace(/^\/api\/v\d+\//, "").split("/").filter(Boolean);
    const namespace = pathParts[0] || "resource";
    const object = pathParts[1] || "default";
    const relation = String(action).toUpperCase() === "GET" ? "viewer" : "editor";

    let nexusAllowed = false;
    let nexusChecked = false;
    let nexusPath = `${subject} -> ${relation} -> ${namespace}:${object}`;
    let nexusError: string | undefined;

    try {
      const nexusPayload = {
        namespace,
        object,
        relation,
        subject: cleanSubjectId,
        subject_namespace: "user",
        subject_id: cleanSubjectId,
        context,
      };

      const nexusRes = await fetch(`${getServiceUrl("nexus")}/check`, {
        method: "POST",
        headers: buildInternalHeaders(true),
        body: JSON.stringify(nexusPayload),
        cache: "no-store",
        signal: AbortSignal.timeout(EXPLORER_TIMEOUT_MS),
      });

      if (nexusRes.ok) {
        nexusChecked = true;
        const nData = await nexusRes.json();
        nexusAllowed = Boolean(nData.allowed);
        if (nData.path) nexusPath = String(nData.path);
      } else {
        nexusAllowed = false;
        nexusError = `Nexus returned HTTP ${nexusRes.status}`;
      }
    } catch {
      nexusAllowed = false;
      nexusError = "Nexus engine unreachable";
    }

    // 3. Evaluate Themis ABAC Policy (fail closed by default)
    let themisAllowed = false;
    let themisEvaluated = false;
    let themisPoliciesMatched = 0;
    let failedPolicy: string | undefined;
    let themisError: string | undefined;

    try {
      const themisPayload = {
        tenant_id: "default",
        payload: {
          subject: cleanSubjectId,
          resource,
          action,
          now: new Date().toISOString(),
          ...context,
        },
      };

      const themisRes = await fetch(`${getServiceUrl("themis")}/policies/evaluate`, {
        method: "POST",
        headers: buildInternalHeaders(true),
        body: JSON.stringify(themisPayload),
        cache: "no-store",
        signal: AbortSignal.timeout(EXPLORER_TIMEOUT_MS),
      });

      if (themisRes.ok) {
        themisEvaluated = true;
        const tData = await themisRes.json();
        themisAllowed = Boolean(tData.all_passed === true);
        themisPoliciesMatched = typeof tData.total_evaluated === "number" ? tData.total_evaluated : 0;
        if (!themisAllowed && Array.isArray(tData.results)) {
          const failed = (tData.results as Record<string, unknown>[]).find((r) => !r.passed);
          if (failed) failedPolicy = String(failed.policy_name || failed.policy_id || "guardrail");
        }
      } else {
        themisAllowed = false;
        themisError = `Themis returned HTTP ${themisRes.status}`;
      }
    } catch {
      themisAllowed = false;
      themisError = "Themis engine unreachable";
    }

    const overallAllowed = aegisMatched && nexusAllowed && themisAllowed;

    let reason = "Access granted across all Zero-Trust evaluation engines.";
    if (!aegisMatched) {
      reason = aegisError
        ? `Denied (fail-closed): ${aegisError}.`
        : "Denied by Aegis: No matching enabled proxy ingress rule found for path.";
    } else if (!nexusAllowed) {
      reason = nexusError
        ? `Denied (fail-closed): ${nexusError}.`
        : `Denied by Nexus: Subject does not possess '${relation}' relation on ${namespace}:${object}.`;
    } else if (!themisAllowed) {
      reason = themisError
        ? `Denied (fail-closed): ${themisError}.`
        : `Denied by Themis: Policy '${failedPolicy || "guardrail"}' evaluated to false.`;
    }

    const result: EffectiveAccessResult = {
      subject,
      resource,
      action,
      allowed: overallAllowed,
      reason,
      engine_breakdown: {
        aegis: {
          matched: aegisMatched,
          rule_id: aegisRuleId,
          rule_name: aegisRuleName,
          upstream,
        },
        nexus: {
          checked: nexusChecked,
          namespace,
          object,
          relation,
          allowed: nexusAllowed,
          path: nexusPath,
        },
        themis: {
          evaluated: themisEvaluated,
          policies_matched: themisPoliciesMatched,
          allowed: themisAllowed,
          failed_policy: failedPolicy,
        },
      },
    };

    return NextResponse.json(result);
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Effective access calculation failed";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
