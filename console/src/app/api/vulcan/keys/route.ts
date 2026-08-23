import { NextResponse } from "next/server";
import { CreateKeySchema, RootKeyRecord, KeyMetadata } from "@/lib/schemas/vulcan";
import { vulcanDb } from "@/lib/server/db";
import { generateRootSecret } from "@/lib/server/macaroons";
import crypto from "crypto";

// GET /api/vulcan/keys - List all keys (metadata only)
export async function GET() {
  const keys = vulcanDb.getKeys();
  // Strip secrets before sending to client
  const safeKeys: KeyMetadata[] = keys.map((k) => { const metadata = { id: k.id, name: k.name, prefix: k.prefix, scopes: k.scopes, created_at: k.created_at, expires_at: k.expires_at, revoked: k.revoked }; return metadata; });
  // Sort by newest first
  safeKeys.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
  
  return NextResponse.json(safeKeys);
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
    
    // Calculate expiration
    let expires_at = null;
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
    
    vulcanDb.addKey(newKey);
    
    const metadata = { id: newKey.id, name: newKey.name, prefix: newKey.prefix, scopes: newKey.scopes, created_at: newKey.created_at, expires_at: newKey.expires_at, revoked: newKey.revoked };
    
    // Return both metadata and the plaintext secret ONCE
    return NextResponse.json({ metadata, secret: fullSecret }, { status: 201 });
    
  } catch {
    return NextResponse.json({ error: "Internal Server Error" }, { status: 500 });
  }
}
