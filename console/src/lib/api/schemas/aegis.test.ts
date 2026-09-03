import { describe, expect, it } from "vitest";
import { ruleListSchema, ruleSchema, testMatchResponseSchema } from "./aegis";

describe("ruleSchema", () => {
  const validRule = {
    id: "rule-1",
    description: "Protect admin API",
    match: { url: "/admin/<.*>", methods: ["GET", "POST"] },
    authenticators: [{ handler: "bearer_token", config: {} }],
    authorizer: { handler: "allow", config: {} },
    mutators: [{ handler: "header", config: { name: "X-User" } }],
    upstream: { url: "http://admin-service:8080" },
  };

  it("accepts a realistic rule payload", () => {
    expect(ruleSchema.safeParse(validRule).success).toBe(true);
  });

  it("accepts a list response", () => {
    expect(ruleListSchema.safeParse([validRule]).success).toBe(true);
  });

  it("rejects a rule missing match.methods", () => {
    const payload = { ...validRule, match: { url: "/admin/<.*>" } };
    expect(ruleSchema.safeParse(payload).success).toBe(false);
  });
});

describe("testMatchResponseSchema", () => {
  it("accepts an unmatched response", () => {
    expect(testMatchResponseSchema.safeParse({ matched: false }).success).toBe(true);
  });

  it("rejects a response missing matched", () => {
    expect(testMatchResponseSchema.safeParse({ rule: null }).success).toBe(false);
  });
});

describe("aegis phase 6 schema extensions", () => {
  it("accepts a rule with order_idx, strip_prefix and rewrite", () => {
    const extendedRule = {
      id: "rule-advanced",
      description: "Advanced routing with rewrite",
      order_idx: 42,
      match: { url: "/api/v2/<.*>", methods: ["GET"] },
      authenticators: [{ handler: "jwt", config: { jwks_url: "http://janus:4444/.well-known/jwks.json" } }],
      authorizer: { handler: "themis", config: { policy_id: "pol-1" } },
      mutators: [],
      upstream: {
        url: "http://target-backend:8080",
        strip_prefix: "/api/v2",
        rewrite: "/internal/v1/$1",
      },
    };
    const result = ruleSchema.safeParse(extendedRule);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.order_idx).toBe(42);
      expect(result.data.upstream.strip_prefix).toBe("/api/v2");
      expect(result.data.upstream.rewrite).toBe("/internal/v1/$1");
    }
  });

  it("validates handlerCatalogueSchema", async () => {
    const { handlerCatalogueSchema } = await import("./aegis");
    const catalogue = {
      authenticators: [{ name: "jwt", description: "JWT validation", config_schema: { type: "object" } }],
      authorizers: [{ name: "themis", description: "ABAC engine", config_schema: {} }],
      mutators: [{ name: "header", description: "Header mutation", config_schema: {} }],
    };
    expect(handlerCatalogueSchema.safeParse(catalogue).success).toBe(true);
  });

  it("validates pipelineTraceSchema and enhanced testMatchResponseSchema", async () => {
    const { pipelineTraceSchema, testMatchResponseSchema } = await import("./aegis");
    const trace = {
      matched_rule_id: "rule-1",
      final_verdict: "allow",
      steps: [
        { stage: "match", status: "success", details: "matched URL" },
        {
          stage: "authenticator",
          handler: "jwt",
          status: "success",
          session: { subject: "usr-123", scopes: ["read"], extra: {}, headers: {} },
        },
        { stage: "authorizer", handler: "allow", status: "success", allowed: true },
        { stage: "mutator", handler: "header", status: "success", mutated_headers: { "x-user": ["usr-123"] } },
        { stage: "upstream", status: "success", target_url: "http://upstream:8080/path" },
      ],
    };
    expect(pipelineTraceSchema.safeParse(trace).success).toBe(true);

    const testMatchWithTrace = {
      matched: true,
      rule: {
        id: "rule-1",
        match: { url: "/admin/<.*>", methods: ["GET"] },
        authenticators: [],
        authorizer: { handler: "allow" },
        mutators: [],
        upstream: { url: "http://upstream:8080" },
      },
      trace,
    };
    expect(testMatchResponseSchema.safeParse(testMatchWithTrace).success).toBe(true);
  });

  it("validates ruleVersionSchema and ruleVersionListSchema", async () => {
    const { ruleVersionSchema, ruleVersionListSchema } = await import("./aegis");
    const version = {
      version: 3,
      description: "Snapshot before update",
      rules: [
        {
          id: "rule-1",
          match: { url: "/admin/<.*>", methods: ["GET"] },
          authenticators: [],
          authorizer: { handler: "allow" },
          mutators: [],
          upstream: { url: "http://upstream:8080" },
        },
      ],
      created_at: "2026-09-02T20:00:00Z",
    };
    expect(ruleVersionSchema.safeParse(version).success).toBe(true);
    expect(ruleVersionListSchema.safeParse([version]).success).toBe(true);
  });

  it("validates reorderRulesRequestSchema", async () => {
    const { reorderRulesRequestSchema } = await import("./aegis");
    expect(reorderRulesRequestSchema.safeParse({ ids: ["rule-1", "rule-2"] }).success).toBe(true);
    expect(reorderRulesRequestSchema.safeParse({ ids: "not-an-array" }).success).toBe(false);
  });
});
