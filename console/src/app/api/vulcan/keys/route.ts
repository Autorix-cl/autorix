import { NextResponse } from "next/server";
import { z } from "zod";
import { CreateKeySchema } from "@/lib/schemas/vulcan";
import { proxyRequest } from "@/lib/api/proxy";

function stripKeySecrets(record: Record<string, unknown>): Record<string, unknown> {
  const sanitized = { ...record };
  delete sanitized.secret;
  delete sanitized.raw_token;
  delete sanitized.root_signature_key;
  delete sanitized.prev_root_signature_key;
  delete sanitized.key_hash;
  return sanitized;
}

// GET /api/vulcan/keys - List all keys without ever exposing raw secrets
export async function GET(req: Request) {
  const url = new URL(req.url);
  const cursor = url.searchParams.get("cursor") || "";
  const limit = url.searchParams.get("limit") || "100";

  const qs = new URLSearchParams();
  if (cursor) qs.set("cursor", cursor);
  if (limit) qs.set("limit", limit);

  const res = await proxyRequest("vulcan", `/keys?${qs.toString()}`, z.any());
  if (!res.ok) {
    return res;
  }

  const data = await res.json();
  const list = Array.isArray(data) ? data : data.data || [];
  const safeList = list.map((item: unknown) =>
    item && typeof item === "object" ? stripKeySecrets(item as Record<string, unknown>) : item,
  );
  return NextResponse.json(safeList);
}

// POST /api/vulcan/keys - Create a new root key via Vulcan engine
export async function POST(req: Request) {
  let body: unknown;
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "Invalid JSON payload" }, { status: 400 });
  }

  const result = CreateKeySchema.safeParse(body);
  if (!result.success) {
    return NextResponse.json({ error: "Invalid input", details: result.error.format() }, { status: 400 });
  }

  const data = result.data;

  let expires_at: string | undefined;
  if (data.expires_in !== "never") {
    const days = parseInt(data.expires_in.replace("d", ""), 10);
    const date = new Date();
    date.setDate(date.getDate() + days);
    expires_at = date.toISOString();
  }

  const proxyPayload = {
    name: data.name,
    owner_id: "system",
    scopes: data.scopes,
    is_live: true,
    expires_at,
  };

  const res = await proxyRequest("vulcan", "/keys", z.any(), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(proxyPayload),
  });

  if (!res.ok) {
    return res;
  }

  const resData = await res.json();
  return NextResponse.json(
    {
      metadata: {
        id: resData.api_key?.id,
        name: resData.api_key?.name ?? data.name,
        prefix: (resData.api_key?.key_prefix || "av_live_") + (resData.api_key?.key_hint || "****"),
        scopes: resData.api_key?.scopes ?? data.scopes,
        created_at: resData.api_key?.created_at ?? new Date().toISOString(),
        expires_at: resData.api_key?.expires_at ?? expires_at ?? null,
        revoked: resData.api_key?.state === "revoked",
      },
      secret: resData.raw_token,
      macaroon: resData.macaroon,
    },
    { status: 201 },
  );
}

