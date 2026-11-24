import { NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";

export async function GET(req: Request) {
  const url = new URL(req.url);

  const tenantId = url.searchParams.get("tenant_id") || "default";
  const limit = url.searchParams.get("limit") || "100";
  const cursor = url.searchParams.get("cursor") || "";

  const qs = new URLSearchParams();
  qs.set("tenant_id", tenantId);
  if (limit) qs.set("limit", limit);
  if (cursor) qs.set("cursor", cursor);

  const res = await proxyRequest("themis", `/policies?${qs.toString()}`, z.any());
  if (!res.ok) return res;

  const json = await res.json();
  const list = Array.isArray(json) ? json : json.data || [];
  return NextResponse.json(list);
}

export async function POST(req: Request) {
  try {
    const body = await req.json();
    const { name, expression, priority } = body;

    if (!name || typeof name !== "string" || !expression || typeof expression !== "string") {
      return NextResponse.json({ error: "Missing required fields" }, { status: 400 });
    }

    const payload = {
      tenant_id: body.tenant_id || body.tenantId || "default",
      name,
      description: body.description || "",
      expression,
      priority: Number(priority ?? 1),
      enabled: body.enabled !== false,
      labels: body.labels || {},
    };

    return proxyRequest("themis", "/policies", z.any(), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to create policy";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
