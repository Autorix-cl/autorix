import { NextResponse } from "next/server";
import { vulcanDb } from "@/lib/server/db";
import { createMacaroon, addCaveat, serializeMacaroon } from "@/lib/server/macaroons";
import { z } from "zod";

const IssueSchema = z.object({
  keyId: z.string(),
  secret: z.string(),
  caveats: z.array(z.string()),
});

// POST /api/vulcan/macaroons/attenuate
// Issues a new Macaroon from a root key and adds caveats. 
// In a real production environment, attenuation usually happens offline on the client.
// This endpoint is for the "Attenuation Studio" to easily generate tokens.
export async function POST(req: Request) {
  try {
    const body = await req.json();
    const result = IssueSchema.safeParse(body);
    
    if (!result.success) {
      return NextResponse.json({ error: "Invalid input" }, { status: 400 });
    }
    
    const { keyId, secret, caveats } = result.data;
    
    // Validate root key exists
    const rootKey = vulcanDb.getKeyById(keyId);
    if (!rootKey) {
      return NextResponse.json({ error: "Root key not found" }, { status: 404 });
    }
    
    if (rootKey.revoked) {
      return NextResponse.json({ error: "Root key is revoked" }, { status: 403 });
    }
    
    // Verify secret matches what we have in DB
    if (rootKey.secret !== secret) {
      return NextResponse.json({ error: "Invalid root secret" }, { status: 401 });
    }
    
    // Generate base macaroon
    let macaroon = createMacaroon(keyId, secret);
    
    // Add all caveats
    for (const caveat of caveats) {
      macaroon = addCaveat(macaroon, caveat);
    }
    
    const token = serializeMacaroon(macaroon);
    
    return NextResponse.json({ 
      macaroon,
      token 
    }, { status: 200 });
    
  } catch {
    return NextResponse.json({ error: "Internal Server Error" }, { status: 500 });
  }
}
