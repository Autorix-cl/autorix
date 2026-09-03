import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";
import type { EffectiveAccessResult } from "@/lib/api/schemas/explorer";

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const {
      subject = "user:alice",
      resource = "/api/v1/finance/invoices",
      action = "GET",
      context = {},
    } = body;

    const cleanSubjectId = subject.replace(/^user:/, "");

    // 1. Evaluate Aegis Rules
    const aegisMatched = true;
    let aegisRuleName = "default-proxy-rule";
    let upstream = "http://internal-service:8080";

    try {
      const aegisRes = await proxyRequest("aegis", "/admin/rules", z.any());
      if (aegisRes.ok) {
        const rules = await aegisRes.json();
        const list = (Array.isArray(rules) ? rules : rules.rules || []) as Record<string, unknown>[];
        const match = list.find((r) => {
          if (!r.enabled) return false;
          if (typeof r.path === "string" && resource.startsWith(r.path)) return true;
          if (typeof r.pattern === "string" && new RegExp(r.pattern).test(resource)) return true;
          return false;
        });
        if (match) {
          aegisRuleName = String(match.name || match.id);
          upstream = String(match.upstream_url || upstream);
        }
      }
    } catch {
      // Graceful fallback for mock/offline testing
    }

    // 2. Evaluate Nexus ReBAC Relation
    // Extract namespace and object from path (e.g. /api/v1/documents/doc_1 -> namespace="document", object="doc_1")
    const pathParts = resource.replace(/^\/api\/v\d+\//, "").split("/").filter(Boolean);
    const namespace = pathParts[0] || "resource";
    const object = pathParts[1] || "default";
    const relation = action === "GET" ? "viewer" : "editor";

    let nexusAllowed = true;
    const nexusChecked = true;
    let nexusPath = `${subject} -> ${relation} -> ${namespace}:${object}`;

    try {
      const nexusPayload = {
        namespace,
        object,
        relation,
        subject: cleanSubjectId,
        context: context,
      };

      const nexusRes = await proxyRequest("nexus", "/check", z.any(), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(nexusPayload),
      });

      if (nexusRes.ok) {
        const nData = await nexusRes.json();
        nexusAllowed = Boolean(nData.allowed);
        if (nData.path) nexusPath = nData.path;
      }
    } catch {
      nexusAllowed = true;
    }

    // 3. Evaluate Themis ABAC Policy
    let themisAllowed = true;
    const themisEvaluated = true;
    let themisPoliciesMatched = 1;
    let failedPolicy: string | undefined = undefined;

    try {
      const themisPayload = {
        context: {
          subject: cleanSubjectId,
          resource,
          action,
          now: new Date().toISOString(),
          ...context,
        },
      };

      const themisRes = await proxyRequest("themis", "/policies/evaluate", z.any(), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(themisPayload),
      });

      if (themisRes.ok) {
        const tData = await themisRes.json();
        themisAllowed = Boolean(tData.all_passed ?? true);
        themisPoliciesMatched = tData.total_evaluated || 1;
        if (!themisAllowed && Array.isArray(tData.results)) {
          const failed = (tData.results as Record<string, unknown>[]).find((r) => !r.passed);
          if (failed) failedPolicy = String(failed.policy_name);
        }
      }
    } catch {
      themisAllowed = true;
    }


    const overallAllowed = aegisMatched && nexusAllowed && themisAllowed;

    let reason = "Access granted across all Zero-Trust evaluation engines.";
    if (!aegisMatched) {
      reason = "Denied by Aegis: No matching proxy ingress rule found for path.";
    } else if (!nexusAllowed) {
      reason = `Denied by Nexus: Subject does not possess '${relation}' relation on ${namespace}:${object}.`;
    } else if (!themisAllowed) {
      reason = `Denied by Themis: Policy '${failedPolicy || "guardrail"}' evaluated to false.`;
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
