import { NextRequest, NextResponse } from "next/server";
import crypto from "crypto";
import type { RequestSimulationTrace, SimulationTraceStep } from "@/lib/api/schemas/explorer";

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const {
      method = "GET",
      path = "/api/v1/projects/proj_alpha/deployments",
      headers = {},
      subject = "user:usr_operator",
    } = body;

    const traceId = `sim_${crypto.randomBytes(6).toString("hex")}`;
    const authHeader = headers["authorization"] || headers["Authorization"] || "";

    const steps: SimulationTraceStep[] = [];
    const isAllowed = true;

    // Step 1: Aegis Route Matching
    steps.push({
      step: "Ingress & Route Pipeline Matching",
      engine: "aegis",
      status: "pass",
      latency_ms: 0.9,
      details: {
        matched_route: "/api/v1/projects/*",
        upstream: "http://deploy-engine.internal:8080",
        strip_prefix: false,
      },
    });

    // Step 2: Authentication Resolution
    let authEngine: "ego" | "janus" | "vulcan" = "ego";
    let credentialType = "Session Cookie";
    if (authHeader.startsWith("Bearer av_live_") || authHeader.startsWith("av_live_")) {
      authEngine = "vulcan";
      credentialType = "Vulcan API Key / Macaroon";
    } else if (authHeader.startsWith("Bearer ey")) {
      authEngine = "janus";
      credentialType = "Janus OAuth2 JWT";
    }

    steps.push({
      step: "Authentication Credential Resolution",
      engine: authEngine,
      status: "pass",
      latency_ms: 1.4,
      details: {
        credential_type: credentialType,
        resolved_subject: subject,
        mfa_verified: true,
      },
    });

    // Step 3: Nexus Zanzibar ReBAC Check
    const pathParts = path.replace(/^\/api\/v\d+\//, "").split("/").filter(Boolean);
    const namespace = pathParts[0] || "project";
    const object = pathParts[1] || "proj_alpha";
    const relation = method === "GET" ? "viewer" : "operator";

    steps.push({
      step: "Zanzibar ReBAC Relation Traversal",
      engine: "nexus",
      status: "pass",
      latency_ms: 2.2,
      details: {
        check: `${subject} possesses '${relation}' on ${namespace}:${object}`,
        graph_path: `${subject} -> member -> team:devops#member -> ${namespace}:${object}#${relation}`,
      },
    });

    // Step 4: Themis CEL ABAC Guardrails
    steps.push({
      step: "ABAC Context Policy Guardrail",
      engine: "themis",
      status: "pass",
      latency_ms: 0.8,
      details: {
        evaluated_policies: ["require-authenticated-subject", "allow-devops-in-region"],
        cel_expression: "request.auth.subject != '' && request.time.hours >= 0",
        all_passed: true,
      },
    });

    // Step 5: Upstream Decision
    const totalLatency = steps.reduce((sum, s) => sum + s.latency_ms, 0);

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
        status_code: 200,
        final_decision: "Forwarded to Upstream (HTTP 200 OK)",
        latency_total_ms: Math.round(totalLatency * 10) / 10,
      },
    };

    return NextResponse.json(trace);
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Simulation failed";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
