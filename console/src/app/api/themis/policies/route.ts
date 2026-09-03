import { NextResponse } from "next/server";
import { z } from "zod";
import jsep from "jsep";
import { proxyRequest } from "@/lib/api/proxy";
import { themisDb } from "@/lib/server/themis-db";

export async function GET(req: Request) {
  const localPolicies = themisDb.getPolicies();
  if (localPolicies && localPolicies.length > 0) {
    return NextResponse.json(localPolicies, { status: 200 });
  }

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

    if (!name || !expression) {
      return NextResponse.json({ error: "Missing required fields" }, { status: 400 });
    }

    // Validate CEL syntax before saving
    try {
      jsep(expression);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      return NextResponse.json({ error: "Invalid CEL syntax: " + msg }, { status: 400 });
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

    const createdInDb = {
      id: "pol_" + Math.random().toString(36).substring(2),
      ...payload,
    };
    themisDb.addPolicy(createdInDb);

    const res = await proxyRequest("themis", "/policies", z.any(), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });

    if (res.ok) {
      return res;
    }

    return NextResponse.json(createdInDb, { status: 201 });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to create policy";
    return NextResponse.json({ error: message }, { status: 500 });
  }
}


