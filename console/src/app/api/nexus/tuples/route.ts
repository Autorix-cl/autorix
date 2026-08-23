import { NextResponse } from "next/server";

let tuples = [
  { id: "t1", objectType: "document", objectId: "doc_123", relation: "viewer", subjectType: "user", subjectId: "alice" },
  { id: "t2", objectType: "document", objectId: "doc_123", relation: "editor", subjectType: "user", subjectId: "bob" },
  { id: "t3", objectType: "organization", objectId: "org_1", relation: "member", subjectType: "group", subjectId: "devs", subjectRelation: "member" },
];

export async function GET() {
  return NextResponse.json({ tuples });
}

export async function POST(request: Request) {
  try {
    const body = await request.json();
    const newTuple = {
      id: Math.random().toString(36).substring(7),
      ...body
    };
    tuples.push(newTuple);
    return NextResponse.json({ success: true, tuple: newTuple }, { status: 201 });
  } catch {
    return NextResponse.json({ error: "Failed to add tuple" }, { status: 400 });
  }
}

export async function DELETE(request: Request) {
  try {
    const { searchParams } = new URL(request.url);
    const ids = searchParams.get("ids")?.split(",") || [];
    
    tuples = tuples.filter(t => !ids.includes(t.id));
    return NextResponse.json({ success: true, deleted: ids.length });
  } catch {
    return NextResponse.json({ error: "Failed to delete tuples" }, { status: 400 });
  }
}
