/* eslint-disable */
import { NextResponse } from "next/server";
import { themisDb } from "@/lib/server/themis-db";
import { evalCel } from "@/lib/server/themis-evaluator";

export async function POST(req: Request) {
  try {
    const body = await req.json();
    const { context, expression } = body;
    
    if (!context) {
      return NextResponse.json({ error: "Missing context" }, { status: 400 });
    }
    
    // Ad-hoc dry run evaluation
    if (expression) {
      const allowed = evalCel(expression, context);
      return NextResponse.json({
        result: allowed ? "Passed" : "Failed",
        matchedPolicy: null
      });
    }
    
    // Full evaluation against enabled policies
    const policies = themisDb.getPolicies().filter((p: any) => p.enabled);
    
    for (const policy of policies) {
      const allowed = evalCel(policy.expression, context);
      if (allowed) {
        return NextResponse.json({
          result: "Passed",
          matchedPolicy: policy
        });
      }
    }
    
    // Default deny if no policy matched
    return NextResponse.json({
      result: "Failed",
      matchedPolicy: null
    });
    
  } catch (err: any) {
    return NextResponse.json({ error: err.message }, { status: 500 });
  }
}
