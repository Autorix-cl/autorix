import { NextRequest, NextResponse } from "next/server";
import crypto from "crypto";
import { getServiceUrl } from "@/lib/api-config";
import { requireOperator } from "@/lib/auth/session";
import type { RequestSimulationTrace, SimulationTraceStep } from "@/lib/api/schemas/explorer";

const SIMULATE_TIMEOUT_MS = 8_000;

function roundMs(ms: number): number {
  return Math.round(ms * 10) / 10;
}

function internalHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    "x-autorix-caller": "console",
  };
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
      method = "GET",
      path = "/api/v1/projects/proj_alpha/deployments",
      headers = {},
      subject = "user:usr_operator",
    } = body;

    const cleanSubjectId = String(subject).replace(/^user:/, "");
    const traceId = `sim_${crypto.randomBytes(6).toString("hex")}`;
    const authHeader = headers["authorization"] || headers["Authorization"] || "";

    const steps: SimulationTraceStep[] = [];
    let anyUnreachable = false;

    // Step 1: Real Aegis Route Matching via POST /rules/test-match
    let aegisPassed = false;
    const aegisStart = performance.now();
    try {
      const aegisRes = await fetch(`${getServiceUrl("aegis")}/rules/test-match`, {
        method: "POST",
        headers: internalHeaders(),
        body: JSON.stringify({ method: String(method).toUpperCase(), url: path, path, headers }),
        cache: "no-store",
        signal: AbortSignal.timeout(SIMULATE_TIMEOUT_MS),
      });
      const aegisLatency = roundMs(performance.now() - aegisStart);
      if (aegisRes.ok) {
        const aegisData = (await aegisRes.json().catch(() => ({}))) as Record<string, unknown>;
        const matched = Boolean(aegisData.matched ?? aegisData.rule ?? aegisData.rule_id);
        const verdict = typeof aegisData.final_verdict === "string" ? aegisData.final_verdict : undefined;
        aegisPassed = matched && verdict !== "deny" && verdict !== "unauthorized" && verdict !== "error";
        steps.push({
          step: "Ingress & Route Pipeline Matching",
          engine: "aegis",
          status: aegisPassed ? "pass" : "fail",
          latency_ms: aegisLatency,
          details: {
            matched,
            rule: aegisData.rule_id || aegisData.rule_name || aegisData.matched_rule || null,
            final_verdict: verdict ?? (aegisPassed ? "allow" : "deny"),
          },
        });
      } else {
        steps.push({
          step: "Ingress & Route Pipeline Matching",
          engine: "aegis",
          status: "fail",
          latency_ms: aegisLatency,
          details: { error: `Aegis returned HTTP ${aegisRes.status}` },
        });
      }
    } catch (err) {
      anyUnreachable = true;
      steps.push({
        step: "Ingress & Route Pipeline Matching",
        engine: "aegis",
        status: "fail",
        latency_ms: roundMs(performance.now() - aegisStart),
        details: { error: err instanceof Error ? err.message : "Aegis engine unreachable" },
      });
    }

    // Step 2: Authentication Credential Classification
    let authEngine: "ego" | "janus" | "vulcan" = "ego";
    let credentialType = "Session Cookie";
    if (authHeader.startsWith("Bearer av_live_") || authHeader.startsWith("av_live_")) {
      authEngine = "vulcan";
      credentialType = "Vulcan API Key / Macaroon";
    } else if (authHeader.startsWith("Bearer ey")) {
      authEngine = "janus";
      credentialType = "Janus OAuth2 JWT";
    }

    const hasSubject = Boolean(cleanSubjectId);
    steps.push({
      step: "Authentication Credential Resolution",
      engine: authEngine,
      status: hasSubject ? "pass" : "fail",
      latency_ms: 0,
      details: {
        credential_type: credentialType,
        resolved_subject: subject,
        simulated_identity: true,
      },
    });

    // Step 3: Real Nexus Zanzibar ReBAC Check via POST /check
    const pathParts = String(path).replace(/^\/api\/v\d+\//, "").split("/").filter(Boolean);
    const namespace = pathParts[0] || "project";
    const object = pathParts[1] || "default";
    const relation = String(method).toUpperCase() === "GET" ? "viewer" : "operator";

    let nexusPassed = false;
    const nexusStart = performance.now();
    try {
      const nexusRes = await fetch(`${getServiceUrl("nexus")}/check`, {
        method: "POST",
        headers: internalHeaders(),
        body: JSON.stringify({
          namespace,
          object,
          relation,
          subject: cleanSubjectId,
          subject_namespace: "user",
          subject_id: cleanSubjectId,
        }),
        cache: "no-store",
        signal: AbortSignal.timeout(SIMULATE_TIMEOUT_MS),
      });
      const nexusLatency = roundMs(performance.now() - nexusStart);
      if (nexusRes.ok) {
        const nexusData = (await nexusRes.json().catch(() => ({}))) as Record<string, unknown>;
        nexusPassed = Boolean(nexusData.allowed);
        steps.push({
          step: "Zanzibar ReBAC Relation Traversal",
          engine: "nexus",
          status: nexusPassed ? "pass" : "fail",
          latency_ms: nexusLatency,
          details: {
            check: `${subject} possesses '${relation}' on ${namespace}:${object}`,
            reason: nexusData.reason || (nexusPassed ? "allowed" : "denied"),
          },
        });
      } else {
        steps.push({
          step: "Zanzibar ReBAC Relation Traversal",
          engine: "nexus",
          status: "fail",
          latency_ms: nexusLatency,
          details: { error: `Nexus returned HTTP ${nexusRes.status}` },
        });
      }
    } catch (err) {
      anyUnreachable = true;
      steps.push({
        step: "Zanzibar ReBAC Relation Traversal",
        engine: "nexus",
        status: "fail",
        latency_ms: roundMs(performance.now() - nexusStart),
        details: { error: err instanceof Error ? err.message : "Nexus engine unreachable" },
      });
    }

    // Step 4: Real Themis CEL ABAC Evaluation via POST /policies/evaluate
    let themisPassed = false;
    const themisStart = performance.now();
    try {
      const themisRes = await fetch(`${getServiceUrl("themis")}/policies/evaluate`, {
        method: "POST",
        headers: internalHeaders(),
        body: JSON.stringify({
          tenant_id: "default",
          payload: {
            subject: cleanSubjectId,
            method: String(method).toUpperCase(),
            path,
            now: new Date().toISOString(),
          },
        }),
        cache: "no-store",
        signal: AbortSignal.timeout(SIMULATE_TIMEOUT_MS),
      });
      const themisLatency = roundMs(performance.now() - themisStart);
      if (themisRes.ok) {
        const themisData = (await themisRes.json().catch(() => ({}))) as Record<string, unknown>;
        themisPassed = Boolean(themisData.all_passed === true);
        steps.push({
          step: "ABAC Context Policy Guardrail",
          engine: "themis",
          status: themisPassed ? "pass" : "fail",
          latency_ms: themisLatency,
          details: {
            total_evaluated: themisData.total_evaluated ?? 0,
            all_passed: themisPassed,
          },
        });
      } else {
        steps.push({
          step: "ABAC Context Policy Guardrail",
          engine: "themis",
          status: "fail",
          latency_ms: themisLatency,
          details: { error: `Themis returned HTTP ${themisRes.status}` },
        });
      }
    } catch (err) {
      anyUnreachable = true;
      steps.push({
        step: "ABAC Context Policy Guardrail",
        engine: "themis",
        status: "fail",
        latency_ms: roundMs(performance.now() - themisStart),
        details: { error: err instanceof Error ? err.message : "Themis engine unreachable" },
      });
    }

    const totalLatency = roundMs(steps.reduce((sum, s) => sum + s.latency_ms, 0));
    const isAllowed = aegisPassed && hasSubject && nexusPassed && themisPassed;
    const statusCode = isAllowed ? 200 : anyUnreachable ? 503 : 403;
    const finalDecision = isAllowed
      ? "Forwarded to Upstream (HTTP 200 OK)"
      : anyUnreachable
        ? "Denied (Fail-Closed: Required Authorization Engine Unreachable)"
        : "Denied by Zero-Trust Evaluation Pipeline (HTTP 403 Forbidden)";

    const trace: RequestSimulationTrace = {
      trace_id: traceId,
      request: {
        method,
        path,
        headers,
        subject,
      },
      steps,
      outcome: {
        allowed: isAllowed,
        status_code: statusCode,
        final_decision: finalDecision,
        latency_total_ms: totalLatency,
      },
    };

    return NextResponse.json(trace);
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Simulation failed";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
