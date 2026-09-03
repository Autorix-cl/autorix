import type { Rule } from "@/lib/api/schemas/aegis";

export interface ShadowWarning {
  ruleId: string;
  shadowedByRuleId: string;
  reason: string;
}

/**
 * Normalizes URL pattern to compare glob / prefix regex semantics.
 */
function isPatternSuperset(broadPattern: string, specificPattern: string): boolean {
  const broad = broadPattern.trim();
  const specific = specificPattern.trim();

  // 1. Identical patterns
  if (broad === specific) {
    return true;
  }

  // 2. Global wildcard (<.*>, *, or regex .*)
  if (broad === "<.*>" || broad === ".*" || broad === "*") {
    return true;
  }

  // 3. Prefix wildcard (e.g. "/api/<.*>", "/api/*", "http://host/api/<.*>")
  const prefixMatch = broad.match(/^(.*?)(?:<.*>|\*|\.\*)$/);
  if (prefixMatch) {
    const prefix = prefixMatch[1];
    if (specific.startsWith(prefix)) {
      return true;
    }
  }

  return false;
}

/**
 * Checks if methods A is a superset of methods B.
 */
function isMethodsSuperset(methodsA: string[], methodsB: string[]): boolean {
  if (methodsA.length === 0) return true; // empty = any method in some setups
  const setA = new Set(methodsA.map((m) => m.toUpperCase()));
  return methodsB.every((m) => setA.has(m.toUpperCase()));
}

/**
 * Inspects an ordered list of proxy rules and detects rules that will never be
 * evaluated because an earlier rule with a broader or identical match pattern
 * intercepts all matching traffic first.
 */
export function detectShadowedRules(rules: Rule[]): ShadowWarning[] {
  const warnings: ShadowWarning[] = [];

  for (let i = 0; i < rules.length; i++) {
    const current = rules[i];
    for (let j = 0; j < i; j++) {
      const prior = rules[j];

      const methodsOverlap = isMethodsSuperset(prior.match.methods, current.match.methods);
      if (!methodsOverlap) {
        continue;
      }

      if (isPatternSuperset(prior.match.url, current.match.url)) {
        let reason = `Catch-all or broader path pattern in rule '${prior.id}' intercepts requests before '${current.id}' can be evaluated.`;
        if (prior.match.url === current.match.url) {
          reason = `Duplicate identical match criteria in earlier rule '${prior.id}'.`;
        }
        warnings.push({
          ruleId: current.id,
          shadowedByRuleId: prior.id,
          reason,
        });
        break; // Once shadowed by an earlier rule, don't flag multiple times
      }
    }
  }

  return warnings;
}
