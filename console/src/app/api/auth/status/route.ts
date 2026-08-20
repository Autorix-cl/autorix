import { NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";

export async function GET() {
  try {
    const argusUrl = getServiceUrl("argus");
    const res = await fetch(`${argusUrl}/v1/auth/status`, {
      cache: "no-store",
    });

    if (!res.ok) {
      return NextResponse.json(
        { error: "Failed to check authentication status" },
        { status: res.status }
      );
    }

    const data = await res.json();
    return NextResponse.json(data);
  } catch {
    return NextResponse.json(
      { error: "Control plane service unavailable", bootstrapped: false, operators_count: 0 },
      { status: 503 }
    );
  }
}
