import { NextResponse } from "next/server";

export async function POST(request: Request) {
  try {
    const body = await request.json();
    const { subject, relation, object } = body;
    
    // Fake logic for the simulator
    if (subject === "user:alice" && object === "document:doc_123") {
      return NextResponse.json({
        allowed: true,
        trace: [
          `${subject} is ${relation} of ${object} (direct relation)`,
          `permission resolved via graph traversal`
        ]
      });
    }
    
    return NextResponse.json({
      allowed: false,
      trace: ["No relation path found in current graph"]
    });
  } catch {
    return NextResponse.json({ error: "Failed to execute check" }, { status: 400 });
  }
}
