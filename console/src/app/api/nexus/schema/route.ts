import { NextResponse } from "next/server";

// In-memory mock for the BFF layer
let currentSchema = `definition user {}
definition document {
  relation viewer: user
  relation editor: user
  permission view = viewer + editor
  permission edit = editor
}
definition organization {
  relation admin: user
  relation member: user
}`;

export async function GET() {
  return NextResponse.json({ schema: currentSchema });
}

export async function PUT(request: Request) {
  try {
    const body = await request.json();
    if (typeof body.schema === "string") {
      currentSchema = body.schema;
      return NextResponse.json({ success: true, schema: currentSchema });
    }
    return NextResponse.json({ error: "Invalid schema payload" }, { status: 400 });
  } catch {
    return NextResponse.json({ error: "Failed to parse JSON" }, { status: 400 });
  }
}
