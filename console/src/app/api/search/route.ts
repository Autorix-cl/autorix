import { NextResponse } from "next/server";
import { getServiceUrl, JANUS_ADMIN_URL } from "@/lib/api-config";
import { requireOperator } from "@/lib/auth/session";

const SEARCH_TIMEOUT_MS = 5_000;

export interface SearchResultItem {
  id: string;
  type: "identity" | "oauth2_client" | "api_key" | "policy" | "proxy_rule" | "operator";
  title: string;
  subtitle: string;
  arn: string;
  href: string;
}

async function fetchJsonSafe(url: string): Promise<unknown | null> {
  const headers: Record<string, string> = { "x-autorix-caller": "console" };
  if (process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN) {
    headers["x-autorix-console-token"] = process.env.AUTORIX_CONSOLE_INTERNAL_TOKEN;
  }
  const res = await fetch(url, {
    cache: "no-store",
    headers,
    signal: AbortSignal.timeout(SEARCH_TIMEOUT_MS),
  });
  if (!res.ok) return null;
  return res.json();
}

function extractList(data: unknown): Record<string, unknown>[] {
  if (Array.isArray(data)) return data as Record<string, unknown>[];
  if (data && typeof data === "object") {
    const rec = data as Record<string, unknown>;
    if (Array.isArray(rec.data)) return rec.data as Record<string, unknown>[];
    if (Array.isArray(rec.items)) return rec.items as Record<string, unknown>[];
  }
  return [];
}

export async function GET(request: Request) {
  const auth = await requireOperator();
  if (!auth.ok) return auth.response;

  const { searchParams } = new URL(request.url);
  const q = searchParams.get("q")?.trim() || "";

  if (!q) {
    return NextResponse.json({ results: [], query: "", count: 0 });
  }

  const results: SearchResultItem[] = [];
  const lowerQ = q.toLowerCase();

  const [egoRes, vulcanRes, themisRes, janusRes, nexusRes] = await Promise.allSettled([
    fetchJsonSafe(`${getServiceUrl("ego")}/admin/identities?q=${encodeURIComponent(q)}&limit=25`),
    fetchJsonSafe(`${getServiceUrl("vulcan")}/keys?limit=50`),
    fetchJsonSafe(`${getServiceUrl("themis")}/policies?limit=50`),
    fetchJsonSafe(`${JANUS_ADMIN_URL}/admin/clients?limit=50`),
    fetchJsonSafe(`${getServiceUrl("nexus")}/tuples?limit=50`),
  ]);

  // 1. Ego identities
  if (egoRes.status === "fulfilled" && egoRes.value) {
    for (const item of extractList(egoRes.value)) {
      const traits = (item.traits && typeof item.traits === "object" ? item.traits : {}) as Record<string, unknown>;
      const email = typeof traits.email === "string" ? traits.email : typeof item.email === "string" ? item.email : "";
      const rawName = traits.name;
      const name =
        typeof rawName === "string"
          ? rawName
          : rawName && typeof rawName === "object"
            ? `${String((rawName as Record<string, unknown>).first || "")} ${String((rawName as Record<string, unknown>).last || "")}`.trim()
            : typeof item.name === "string"
              ? item.name
              : "";
      const id = String(item.id || "");
      const state = String(item.state || item.status || "active");

      if (
        email.toLowerCase().includes(lowerQ) ||
        name.toLowerCase().includes(lowerQ) ||
        id.toLowerCase().includes(lowerQ)
      ) {
        results.push({
          id,
          type: "identity",
          title: email || name || id,
          subtitle: `Identity (${state})`,
          arn: `arn:autorix:ego:prod:default:identity/${id}`,
          href: `/identities?id=${encodeURIComponent(id)}`,
        });
      }
    }
  }

  // 2. Vulcan API keys
  if (vulcanRes.status === "fulfilled" && vulcanRes.value) {
    for (const item of extractList(vulcanRes.value)) {
      const id = String(item.id || "");
      const name = typeof item.name === "string" ? item.name : "";
      if (name.toLowerCase().includes(lowerQ) || id.toLowerCase().includes(lowerQ)) {
        results.push({
          id,
          type: "api_key",
          title: name || id,
          subtitle: "API Key",
          arn: `arn:autorix:vulcan:prod:default:key/${id}`,
          href: `/vulcan?id=${encodeURIComponent(id)}`,
        });
      }
    }
  }

  // 3. Themis CEL policies
  if (themisRes.status === "fulfilled" && themisRes.value) {
    for (const item of extractList(themisRes.value)) {
      const id = String(item.id ?? item.ID ?? "");
      const name = String(item.name ?? item.Name ?? "");
      const expr = String(item.expression ?? item.Expression ?? "");
      if (
        name.toLowerCase().includes(lowerQ) ||
        expr.toLowerCase().includes(lowerQ) ||
        id.toLowerCase().includes(lowerQ)
      ) {
        results.push({
          id,
          type: "policy",
          title: name || id,
          subtitle: "CEL Policy",
          arn: `arn:autorix:themis:prod:default:policy/${id}`,
          href: `/policies?id=${encodeURIComponent(id)}`,
        });
      }
    }
  }

  // 4. Janus OAuth2 clients
  if (janusRes.status === "fulfilled" && janusRes.value) {
    for (const item of extractList(janusRes.value)) {
      const id = String(item.client_id ?? item.id ?? "");
      const name = String(item.client_name ?? item.name ?? "");
      if (name.toLowerCase().includes(lowerQ) || id.toLowerCase().includes(lowerQ)) {
        results.push({
          id,
          type: "oauth2_client",
          title: name || id,
          subtitle: "OAuth2 Client",
          arn: `arn:autorix:janus:prod:default:client/${id}`,
          href: `/oauth2?client_id=${encodeURIComponent(id)}`,
        });
      }
    }
  }

  // 5. Nexus relation tuples
  if (nexusRes.status === "fulfilled" && nexusRes.value) {
    for (const item of extractList(nexusRes.value)) {
      const ns = String(item.namespace || "");
      const obj = String(item.object || "");
      const rel = String(item.relation || "");
      const sub = `${String(item.subject_namespace || "user")}:${String(item.subject_id || "")}`;
      const descriptor = `${ns}:${obj}#${rel}@${sub}`;
      if (descriptor.toLowerCase().includes(lowerQ)) {
        results.push({
          id: descriptor,
          type: "proxy_rule",
          title: `${ns}:${obj}`,
          subtitle: `Nexus Tuple (${rel} -> ${sub})`,
          arn: `arn:autorix:nexus:prod:default:tuple/${ns}/${obj}`,
          href: `/permissions?namespace=${encodeURIComponent(ns)}`,
        });
      }
    }
  }

  return NextResponse.json({
    query: q,
    results,
    count: results.length,
  });
}
