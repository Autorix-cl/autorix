import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";
import { goPolicySchema } from "@/lib/api/schemas/themis";

export const dynamic = "force-dynamic";

export async function GET(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const tenantId = searchParams.get("tenant_id") || "default";
  const cursor = searchParams.get("cursor") || "";
  const search = searchParams.get("search")?.toLowerCase();

  const query = new URLSearchParams();
  query.set("tenant_id", tenantId);
  if (cursor) query.set("cursor", cursor);
  const limit = searchParams.get("limit") || "50";
  if (search) {
    query.set("limit", "100");
  } else if (limit) {
    query.set("limit", limit);
  }

  const res = await proxyRequest("themis", `/policies?${query.toString()}`, z.any());
  if (!res.ok) return res;

  const json = await res.json();
  const rawList = Array.isArray(json) ? json : json.data || [];
  let data = rawList.map((p: Record<string, unknown>) => ({
    ID: (p.id ?? p.ID) as string,
    TenantID: ((p.tenant_id ?? p.TenantID) as string) ?? "default",
    Name: (p.name ?? p.Name) as string,
    Description: ((p.description ?? p.Description) as string) ?? "",
    Expression: (p.expression ?? p.Expression) as string,
    Priority: (p.priority ?? p.Priority ?? 1) as number,
    Enabled: (p.enabled ?? p.Enabled ?? true) as boolean,
    Labels: (p.labels ?? p.Labels ?? {}) as Record<string, string>,
    CreatedAt: ((p.created_at ?? p.CreatedAt) as string) ?? "",
    UpdatedAt: ((p.updated_at ?? p.UpdatedAt) as string) ?? "",
  }));

  if (search) {
    data = data.filter(
      (p: { Name: string; Expression: string; Description: string }) =>
        p.Name.toLowerCase().includes(search) ||
        p.Expression.toLowerCase().includes(search) ||
        p.Description.toLowerCase().includes(search),
    );
  }

  // Sort newest first
  data.sort(
    (a: { CreatedAt: string }, b: { CreatedAt: string }) =>
      new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime(),
  );

  return NextResponse.json({
    data,
    has_more: json.has_more ?? false,
    cursor: json.cursor,
  });
}

export async function POST(req: NextRequest) {
  const body = await req.json();
  const payload = {
    tenant_id: body.tenantId || "default",
    name: body.name,
    description: body.description || "",
    expression: body.expression,
    priority: Number(body.priority) || 1,
    enabled: body.enabled !== false,
    labels: body.labels || {},
  };

  const res = await proxyRequest("themis", "/policies", goPolicySchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

  if (!res.ok) return res;

  const p = await res.json();
  return NextResponse.json({
    ID: p.id ?? p.ID,
    TenantID: p.tenant_id ?? p.TenantID ?? "default",
    Name: p.name ?? p.Name,
    Description: p.description ?? p.Description ?? "",
    Expression: p.expression ?? p.Expression,
    Priority: p.priority ?? p.Priority ?? 1,
    Enabled: p.enabled ?? p.Enabled ?? true,
    Labels: p.labels ?? p.Labels ?? {},
    CreatedAt: p.created_at ?? p.CreatedAt ?? "",
    UpdatedAt: p.updated_at ?? p.UpdatedAt ?? "",
  });
}
