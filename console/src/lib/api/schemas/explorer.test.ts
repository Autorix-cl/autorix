import { describe, it, expect } from "vitest";
import {
  unifiedSubjectSchema,
  effectiveAccessResultSchema,
  requestSimulationTraceSchema,
  consistencyFindingListSchema,
} from "./explorer";

describe("Explorer Cross-Engine Schemas", () => {
  it("validates unifiedSubjectSchema", () => {
    const payload = {
      id: "usr_alice",
      identity: {
        id: "usr_alice",
        state: "active",
        traits: { email: "alice@autorix.io", role: "admin" },
      },
      sessions: [
        {
          id: "sess_1",
          identity_id: "usr_alice",
          active: true,
          ip_address: "192.168.1.5",
        },
      ],
      relations: [
        {
          namespace: "document",
          object: "quarterly_budget",
          relation: "owner",
          subject: "user:usr_alice",
        },
      ],
      oauth_grants: [
        {
          id: "grant_1",
          client_id: "cli_mobile_app",
          scope: "openid profile",
        },
      ],
      api_keys: [
        {
          id: "k_1",
          name: "CLI Token",
          prefix: "av_live_1234",
          scopes: ["read", "write"],
          call_count: 50,
        },
      ],
      enterprise_linkage: {
        provider_id: "okta-corp",
        external_id: "okta_001",
        email: "alice@autorix.io",
        active: true,
      },
    };

    const res = unifiedSubjectSchema.safeParse(payload);
    expect(res.success).toBe(true);
  });

  it("validates effectiveAccessResultSchema", () => {
    const payload = {
      subject: "user:usr_alice",
      resource: "/api/v1/finance/invoices",
      action: "GET",
      allowed: true,
      reason: "Access permitted by Aegis rule 'finance-api', Nexus relation 'viewer', and Themis policy 'working-hours-only'",
      engine_breakdown: {
        aegis: {
          matched: true,
          rule_id: "rule_1",
          rule_name: "finance-api",
          upstream: "http://finance-srv:8080",
        },
        nexus: {
          checked: true,
          namespace: "finance",
          object: "invoices",
          relation: "viewer",
          allowed: true,
          path: "user:usr_alice -> member -> finance:invoices#viewer",
        },
        themis: {
          evaluated: true,
          policies_matched: 1,
          allowed: true,
        },
      },
    };

    const res = effectiveAccessResultSchema.safeParse(payload);
    expect(res.success).toBe(true);
  });

  it("validates requestSimulationTraceSchema", () => {
    const payload = {
      trace_id: "sim_abc123",
      request: {
        method: "GET",
        path: "/api/v1/orders",
        headers: { authorization: "Bearer av_live_xyz" },
        subject: "user:bob",
      },
      steps: [
        {
          step: "Route Ingress & Pipeline Match",
          engine: "aegis",
          status: "pass",
          latency_ms: 1.2,
          details: { rule: "orders-route" },
        },
        {
          step: "Auth Verification",
          engine: "vulcan",
          status: "pass",
          latency_ms: 0.8,
          details: { key_prefix: "av_live_" },
        },
        {
          step: "Zanzibar ReBAC Check",
          engine: "nexus",
          status: "pass",
          latency_ms: 2.1,
          details: { tuple_found: true },
        },
        {
          step: "ABAC Policy Guardrail",
          engine: "themis",
          status: "pass",
          latency_ms: 0.5,
          details: { expression: "true" },
        },
      ],
      outcome: {
        allowed: true,
        status_code: 200,
        final_decision: "Forwarded to Upstream",
        latency_total_ms: 4.6,
      },
    };

    const res = requestSimulationTraceSchema.safeParse(payload);
    expect(res.success).toBe(true);
  });

  it("validates consistencyFindingListSchema", () => {
    const findings = [
      {
        id: "cf_1",
        severity: "critical",
        category: "nexus",
        title: "Missing Nexus Namespace Reference",
        description: "Aegis rule 'legacy-api' references relation check in namespace 'crm', but 'crm' is not defined in Nexus schema.",
        remediation: "Add 'crm' definition in Nexus schema or update the Aegis handler.",
        remediation_link: "/nexus",
      },
    ];

    const res = consistencyFindingListSchema.safeParse(findings);
    expect(res.success).toBe(true);
  });
});
