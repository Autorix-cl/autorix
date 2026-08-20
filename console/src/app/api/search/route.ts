import { NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";

export interface SearchResultItem {
  id: string;
  type: "identity" | "oauth2_client" | "api_key" | "policy" | "proxy_rule" | "operator";
  title: string;
  subtitle: string;
  arn: string;
  href: string;
}

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const q = searchParams.get("q")?.trim() || "";

  if (!q) {
    return NextResponse.json({ results: [], query: "" });
  }

  const results: SearchResultItem[] = [];
  const lowerQ = q.toLowerCase();

  // Search identities in Ego
  try {
    const egoUrl = getServiceUrl("ego");
    const res = await fetch(`${egoUrl}/identities`, { cache: "no-store" });
    if (res.ok) {
      const data = await res.json();
      const list = Array.isArray(data) ? data : data.items || [];
      for (const item of list) {
        if (
          item.email?.toLowerCase().includes(lowerQ) ||
          item.name?.toLowerCase().includes(lowerQ) ||
          item.id?.includes(lowerQ)
        ) {
          results.push({
            id: item.id,
            type: "identity",
            title: item.email || item.name || item.id,
            subtitle: `Identity (${item.status || "active"})`,
            arn: `arn:autorix:ego:prod:default:identity/${item.id}`,
            href: `/identities?id=${item.id}`,
          });
        }
      }
    }
  } catch {
    // Gracefully handle unreachable engines
  }

  // Search API keys in Vulcan
  try {
    const vulcanUrl = getServiceUrl("vulcan");
    const res = await fetch(`${vulcanUrl}/keys`, { cache: "no-store" });
    if (res.ok) {
      const data = await res.json();
      const list = Array.isArray(data) ? data : data.items || [];
      for (const item of list) {
        if (item.name?.toLowerCase().includes(lowerQ) || item.id?.includes(lowerQ)) {
          results.push({
            id: item.id,
            type: "api_key",
            title: item.name || item.id,
            subtitle: "API Key",
            arn: `arn:autorix:vulcan:prod:default:key/${item.id}`,
            href: `/api-keys?id=${item.id}`,
          });
        }
      }
    }
  } catch {
    // Gracefully handle unreachable engines
  }

  return NextResponse.json({
    query: q,
    results,
    count: results.length,
  });
}
