import { NextResponse } from "next/server";
import { vulcanDb } from "@/lib/server/db";

// DELETE /api/vulcan/keys/:id - Revoke a root key
export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const resolvedParams = await params;
  const id = resolvedParams.id;
  
  if (!id) {
    return NextResponse.json({ error: "Missing ID" }, { status: 400 });
  }

  const success = vulcanDb.revokeKey(id);
  
  if (!success) {
    return NextResponse.json({ error: "Key not found" }, { status: 404 });
  }

  return NextResponse.json({ success: true }, { status: 200 });
}
