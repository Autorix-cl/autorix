import { NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";
import { buildInternalAuthHeaders } from "@/lib/api/internal-headers";
import { requireOperator } from "@/lib/auth/session";
import type { ConsistencyFinding } from "@/lib/api/schemas/explorer";

const CONSISTENCY_TIMEOUT_MS = 8_000;

async function fetchEngineJson(url: string): Promise<unknown | null> {
  const res = await fetch(url, {
    cache: "no-store",
    headers: buildInternalAuthHeaders(),
    signal: AbortSignal.timeout(CONSISTENCY_TIMEOUT_MS),
  });
  if (!res.ok) return null;
  return res.json();
}

export async function GET() {
  const auth = await requireOperator("explorer:read");
  if (!auth.ok) return auth.response;

  const findings: ConsistencyFinding[] = [];

  // Run cross-engine checks in parallel against real admin endpoints
  const [aegisRes, nexusRes, vulcanRes, hermesRes] = await Promise.allSettled([
    fetchEngineJson(`${getServiceUrl("aegis")}/rules`),
    fetchEngineJson(`${getServiceUrl("nexus")}/admin/namespaces`),
    fetchEngineJson(`${getServiceUrl("vulcan")}/keys?limit=100`),
    fetchEngineJson(`${getServiceUrl("hermes")}/admin/saml/providers`),
  ]);

  const aegisData = aegisRes.status === "fulfilled" ? aegisRes.value : null;
  const nexusData = nexusRes.status === "fulfilled" ? nexusRes.value : null;
  const vulcanData = vulcanRes.status === "fulfilled" ? vulcanRes.value : null;
  const hermesData = hermesRes.status === "fulfilled" ? hermesRes.value : null;

  let hasCrossEngineDrift = false;

  // Check 1: Aegis & Nexus Reachability and Cross-Engine Alignment
  if (!aegisData) {
    hasCrossEngineDrift = true;
    findings.push({
      id: "aegis-unreachable",
      severity: "warning",
      category: "aegis",
      title: "Aegis Proxy Rules Unavailable",
      description: "Could not retrieve active ingress routing rules from Aegis to verify Zero-Trust route alignment.",
      remediation: "Verify Aegis admin listener connectivity and inspect engine health.",
      remediation_link: "/proxy-rules",
    });
  }

  if (!nexusData) {
    hasCrossEngineDrift = true;
    findings.push({
      id: "nexus-namespaces-unavailable",
      severity: "warning",
      category: "nexus",
      title: "Nexus Namespaces Unavailable",
      description: "Could not retrieve ReBAC namespace definitions from Nexus /admin/namespaces.",
      remediation: "Verify Nexus REST admin API connectivity and check schema state.",
      remediation_link: "/permissions",
    });
  }

  if (aegisData && nexusData) {
    const aegisList = (
      Array.isArray(aegisData)
        ? aegisData
        : (aegisData as Record<string, unknown>).rules || (aegisData as Record<string, unknown>).data || []
    ) as Record<string, unknown>[];
    const namespaceList = (
      Array.isArray(nexusData) ? nexusData : (nexusData as Record<string, unknown>).data || []
    ) as Record<string, unknown>[];
    const definedNamespaces = new Set(namespaceList.map((ns) => String(ns.name || "")).filter(Boolean));

    const rulesMissingUpstream = aegisList.filter((r) => {
      if (r.enabled === false) return false;
      const upstreamObj = r.upstream as Record<string, unknown> | undefined;
      const upstreamUrl = r.upstream_url || upstreamObj?.url;
      return !upstreamUrl || typeof upstreamUrl !== "string" || !upstreamUrl.trim();
    });

    if (rulesMissingUpstream.length > 0) {
      hasCrossEngineDrift = true;
      findings.push({
        id: "aegis-rules-missing-upstream",
        severity: "critical",
        category: "aegis",
        title: `${rulesMissingUpstream.length} Active Proxy Rule${rulesMissingUpstream.length === 1 ? "" : "s"} Missing Upstream URL`,
        description: `Enabled Aegis rule(s) (${rulesMissingUpstream
          .map((r) => String(r.name || r.id))
          .slice(0, 3)
          .join(", ")}) lack a valid upstream target.`,
        remediation: "Configure an authenticated upstream target URL or disable the incomplete rule.",
        remediation_link: "/proxy-rules",
      });
    }

    const orphanedNamespaces: string[] = [];
    for (const rule of aegisList) {
      if (rule.enabled === false) continue;
      const authorizer = rule.authorizer as Record<string, unknown> | undefined;
      const config = (authorizer?.config as Record<string, unknown> | undefined) || {};
      const referencedNamespace =
        typeof config.namespace === "string"
          ? config.namespace
          : typeof rule.nexus_namespace === "string"
            ? rule.nexus_namespace
            : undefined;
      if (referencedNamespace && !definedNamespaces.has(referencedNamespace)) {
        orphanedNamespaces.push(`${String(rule.name || rule.id)} -> ${referencedNamespace}`);
      }
    }

    if (orphanedNamespaces.length > 0) {
      hasCrossEngineDrift = true;
      findings.push({
        id: "aegis-nexus-orphaned-namespaces",
        severity: "critical",
        category: "nexus",
        title: `${orphanedNamespaces.length} Proxy Rule${orphanedNamespaces.length === 1 ? "" : "s"} Reference Undefined Nexus Namespaces`,
        description: `Cross-engine drift detected: ${orphanedNamespaces.slice(0, 3).join(", ")} reference namespaces not present in Nexus.`,
        remediation: "Create the missing ReBAC namespaces in Nexus or update the Aegis authorizer configuration.",
        remediation_link: "/permissions",
      });
    }
  }

  // Check 2: Stale Vulcan API Keys
  if (vulcanData) {
    const keys = (
      Array.isArray(vulcanData) ? vulcanData : (vulcanData as Record<string, unknown>).data || []
    ) as Record<string, unknown>[];
    const staleKeys = keys.filter((k) => {
      if (k.state === "revoked") return false;
      if (!k.last_used_at) {
        const created = new Date(String(k.created_at)).getTime();
        return Number.isFinite(created) && (Date.now() - created) / (1000 * 60 * 60 * 24) > 14;
      }
      const lastUsed = new Date(String(k.last_used_at)).getTime();
      return Number.isFinite(lastUsed) && (Date.now() - lastUsed) / (1000 * 60 * 60 * 24) > 30;
    });

    if (staleKeys.length > 0) {
      findings.push({
        id: "vulcan-stale-keys",
        severity: "warning",
        category: "vulcan",
        title: `${staleKeys.length} Stale API Key${staleKeys.length === 1 ? "" : "s"} Detected`,
        description: `API keys (${staleKeys
          .map((k) => String(k.name))
          .slice(0, 3)
          .join(", ")}) have had zero traffic in over 30 days. Unused credentials expand the attack surface.`,
        remediation: "Review key usage and revoke stale machine tokens.",
        remediation_link: "/vulcan",
      });
    }
  }

  // Check 3: Hermes IdP Certificates Expiry
  if (hermesData) {
    const providers = (
      Array.isArray(hermesData) ? hermesData : (hermesData as Record<string, unknown>).data || []
    ) as Record<string, unknown>[];
    providers.forEach((p) => {
      const certs = p.certificates as Record<string, unknown>[] | undefined;
      if (certs) {
        certs.forEach((c) => {
          if (c.expired) {
            findings.push({
              id: `hermes-cert-expired-${p.id}`,
              severity: "critical",
              category: "hermes",
              title: `Expired SAML IdP Certificate · ${String(p.display_name || p.id)}`,
              description: `Certificate for ${String(c.subject)} expired on ${new Date(String(c.not_after)).toLocaleDateString()}. Users will experience federation authentication errors.`,
              remediation: "Upload a renewed IdP certificate and test the SAML assertion round trip.",
              remediation_link: "/enterprise",
            });
          } else if (c.expiring_soon) {
            findings.push({
              id: `hermes-cert-soon-${p.id}`,
              severity: "warning",
              category: "hermes",
              title: `SAML IdP Certificate Expiring in ${String(c.days_until_expiry)} Days`,
              description: `Certificate for ${String(c.subject)} will expire soon. Plan rollover to prevent silent authentication outages.`,
              remediation: "Schedule IdP certificate rotation before expiration.",
              remediation_link: "/enterprise",
            });
          }
        });
      }
    });
  }

  // Only emit synchronized info baseline if Aegis and Nexus were verified with zero cross-engine drift
  if (!hasCrossEngineDrift) {
    findings.push({
      id: "aegis-nexus-alignment",
      severity: "info",
      category: "aegis",
      title: "Zero-Trust Route Alignment Synchronized",
      description:
        "All active Aegis routing definitions were verified against upstream targets and Nexus ReBAC namespaces.",
      remediation: "No immediate action required.",
      remediation_link: "/proxy-rules",
    });
  }

  return NextResponse.json(findings);
}
