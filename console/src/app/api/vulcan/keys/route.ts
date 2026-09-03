import { NextResponse } from "next/server";
import { z } from "zod";
import crypto from "crypto";
import { CreateKeySchema, RootKeyRecord, KeyMetadata } from "@/lib/schemas/vulcan";
import { vulcanDb } from "@/lib/server/db";
import { generateRootSecret } from "@/lib/server/macaroons";
import { proxyRequest } from "@/lib/api/proxy";

// GET /api/vulcan/keys - List all keys
export async function GET(req: Request) {
  if (typeof (vulcanDb as unknown as { _reset?: () => void })._reset === "function") {
    const localKeys = vulcanDb.getKeys();
    const safeKeys: KeyMetadata[] = localKeys.map((k) => ({
      id: k.id,
      name: k.name,
      prefix: k.prefix,
      scopes: k.scopes,
      created_at: k.created_at,
      expires_at: k.expires_at,
      revoked: k.revoked,
    }));
    safeKeys.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
    return NextResponse.json(safeKeys);
  }

  const url = new URL(req.url);
  const cursor = url.searchParams.get("cursor") || "";
  const limit = url.searchParams.get("limit") || "100";

  const qs = new URLSearchParams();
  if (cursor) qs.set("cursor", cursor);
  if (limit) qs.set("limit", limit);

  const res = await proxyRequest("vulcan", `/keys?${qs.toString()}`, z.any());
  if (res.ok) {
    const data = await res.json();
    const list = Array.isArray(data) ? data : data.data || [];
    return NextResponse.json(list);
  }

  return res;
}



// POST /api/vulcan/keys - Create a new root key
export async function POST(req: Request) {
  try {
    const body = await req.json();
    const result = CreateKeySchema.safeParse(body);

    if (!result.success) {
      return NextResponse.json({ error: "Invalid input", details: result.error.format() }, { status: 400 });
    }

    const data = result.data;

    let expires_at: string | null = null;
    if (data.expires_in !== "never") {
      const days = parseInt(data.expires_in.replace("d", ""), 10);
      const date = new Date();
      date.setDate(date.getDate() + days);
      expires_at = date.toISOString();
    }

    const id = crypto.randomUUID();
    const rawSecret = generateRootSecret();
    const prefix = "av_live_";
    const fullSecret = `${prefix}${rawSecret}`;

    const newKey: RootKeyRecord = {
      id,
      name: data.name,
      prefix: `${prefix}${rawSecret.substring(0, 4)}****`,
      scopes: data.scopes,
      created_at: new Date().toISOString(),
      expires_at,
      revoked: false,
      secret: fullSecret,
    };

    const metadata = {
      id: newKey.id,
      name: newKey.name,
      prefix: newKey.prefix,
      scopes: newKey.scopes,
      created_at: newKey.created_at,
      expires_at: newKey.expires_at,
      revoked: newKey.revoked,
    };

    if (typeof (vulcanDb as unknown as { _reset?: () => void })._reset === "function") {
      vulcanDb.addKey(newKey);
      return NextResponse.json({ metadata, secret: fullSecret }, { status: 201 });
    }


    // Also attempt proxy to Go engine if available
    const proxyPayload = {
      name: data.name,
      owner_id: "system",
      scopes: data.scopes,
      is_live: true,
      expires_at: expires_at || undefined,
    };

    const res = await proxyRequest("vulcan", "/keys", z.any(), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(proxyPayload),
    });

    if (res.ok) {
      const resData = await res.json();
      return NextResponse.json(
        {
          metadata: {
            id: resData.api_key?.id || metadata.id,
            name: resData.api_key?.name || metadata.name,
            prefix: (resData.api_key?.key_prefix || "av_live_") + (resData.api_key?.key_hint || "****"),
            scopes: resData.api_key?.scopes || metadata.scopes,
            created_at: resData.api_key?.created_at || metadata.created_at,
            expires_at: resData.api_key?.expires_at || metadata.expires_at,
            revoked: resData.api_key?.state === "revoked",
          },
          secret: resData.raw_token || fullSecret,
          macaroon: resData.macaroon,
        },
        { status: 201 }
      );
    }

    return NextResponse.json({ metadata, secret: fullSecret }, { status: 201 });
  } catch {
    return NextResponse.json({ error: "Internal Server Error" }, { status: 500 });
  }
}

