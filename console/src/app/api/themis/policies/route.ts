import { NextResponse } from "next/server";
import { themisDb } from "@/lib/server/themis-db";
import jsep from "jsep";

export async function GET() {
  const policies = themisDb.getPolicies();
  return NextResponse.json(policies, { status: 200 });
}

export async function POST(req: Request) {
  try {
    const body = await req.json();
    const { name, expression, priority } = body;
    
    if (!name || !expression || priority === undefined) {
      return NextResponse.json({ error: "Missing required fields" }, { status: 400 });
    }
    
    // Validate CEL syntax before saving
    try {
      jsep(expression);
    } catch (err: any) {
      return NextResponse.json({ error: "Invalid CEL syntax: " + err.message }, { status: 400 });
    }
    
    const newPolicy = {
      id: Math.random().toString(36).substring(2),
      name,
      expression,
      priority: Number(priority),
      enabled: true
    };
    
    themisDb.addPolicy(newPolicy);
    
    return NextResponse.json(newPolicy, { status: 201 });
  } catch (err) {
    return NextResponse.json({ error: "Internal Server Error" }, { status: 500 });
  }
}
