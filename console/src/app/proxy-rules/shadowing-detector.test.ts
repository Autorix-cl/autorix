import { describe, it, expect } from "vitest";
import { detectShadowedRules } from "./shadowing-detector";
import type { Rule } from "@/lib/api/schemas/aegis";

describe("detectShadowedRules", () => {
  it("returns no warnings for non-overlapping rules", () => {
    const rules: Rule[] = [
      {
        id: "rule-users",
        match: { methods: ["GET"], url: "/api/users" },
        authenticators: [],
        authorizer: { handler: "allow" },
        mutators: [],
        upstream: { url: "http://upstream:8080" },
      },
      {
        id: "rule-orders",
        match: { methods: ["GET"], url: "/api/orders" },
        authenticators: [],
        authorizer: { handler: "allow" },
        mutators: [],
        upstream: { url: "http://upstream:8080" },
      },
    ];

    const warnings = detectShadowedRules(rules);
    expect(warnings).toHaveLength(0);
  });

  it("detects when a wildcard URL rule shadows a subsequent specific rule with same method", () => {
    const rules: Rule[] = [
      {
        id: "catch-all",
        match: { methods: ["GET"], url: "<.*>" },
        authenticators: [],
        authorizer: { handler: "allow" },
        mutators: [],
        upstream: { url: "http://catchall:8080" },
      },
      {
        id: "specific-user",
        match: { methods: ["GET"], url: "/api/v1/users" },
        authenticators: [],
        authorizer: { handler: "allow" },
        mutators: [],
        upstream: { url: "http://users:8080" },
      },
    ];

    const warnings = detectShadowedRules(rules);
    expect(warnings).toHaveLength(1);
    expect(warnings[0].ruleId).toBe("specific-user");
    expect(warnings[0].shadowedByRuleId).toBe("catch-all");
    expect(warnings[0].reason).toContain("Catch-all or broader path pattern");
  });

  it("detects when a prefix wildcard shadows a nested path rule", () => {
    const rules: Rule[] = [
      {
        id: "api-prefix",
        match: { methods: ["GET", "POST"], url: "/api/<.*>" },
        authenticators: [],
        authorizer: { handler: "allow" },
        mutators: [],
        upstream: { url: "http://api:8080" },
      },
      {
        id: "api-nested",
        match: { methods: ["GET"], url: "/api/v1/billing" },
        authenticators: [],
        authorizer: { handler: "allow" },
        mutators: [],
        upstream: { url: "http://billing:8080" },
      },
    ];

    const warnings = detectShadowedRules(rules);
    expect(warnings).toHaveLength(1);
    expect(warnings[0].ruleId).toBe("api-nested");
    expect(warnings[0].shadowedByRuleId).toBe("api-prefix");
  });

  it("does not flag shadowing if methods do not overlap", () => {
    const rules: Rule[] = [
      {
        id: "get-catch-all",
        match: { methods: ["GET"], url: "<.*>" },
        authenticators: [],
        authorizer: { handler: "allow" },
        mutators: [],
        upstream: { url: "http://catchall:8080" },
      },
      {
        id: "post-api",
        match: { methods: ["POST"], url: "/api/submit" },
        authenticators: [],
        authorizer: { handler: "allow" },
        mutators: [],
        upstream: { url: "http://submit:8080" },
      },
    ];

    const warnings = detectShadowedRules(rules);
    expect(warnings).toHaveLength(0);
  });

  it("detects duplicate identical rules", () => {
    const rules: Rule[] = [
      {
        id: "rule-1",
        match: { methods: ["GET"], url: "/health" },
        authenticators: [],
        authorizer: { handler: "allow" },
        mutators: [],
        upstream: { url: "http://service:8080" },
      },
      {
        id: "rule-2",
        match: { methods: ["GET"], url: "/health" },
        authenticators: [],
        authorizer: { handler: "allow" },
        mutators: [],
        upstream: { url: "http://service:8080" },
      },
    ];

    const warnings = detectShadowedRules(rules);
    expect(warnings).toHaveLength(1);
    expect(warnings[0].ruleId).toBe("rule-2");
    expect(warnings[0].shadowedByRuleId).toBe("rule-1");
  });
});
