import { NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";
import { SESSION_COOKIE_NAME } from "@/lib/auth/session";

export async function POST(request: Request) {
  try {
    const body = await request.json();
    const argusUrl = getServiceUrl("argus");

    const res = await fetch(`${argusUrl}/v1/auth/bootstrap`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(body),
    });

    const data = await res.json();
    if (!res.ok) {
      return NextResponse.json(
        { error: data.error || "Bootstrap setup failed" },
        { status: res.status }
      );
    }

    const response = NextResponse.json({
      success: true,
      operator: data.operator,
    });

    // Set secure HttpOnly cookie for session (P3-S2-T2)
    response.cookies.set({
      name: SESSION_COOKIE_NAME,
      value: data.session_token,
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      expires: new Date(data.expires_at),
    });

    return response;
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : "Internal server error";
    return NextResponse.json(
      { error: message },
      { status: 500 }
    );
  }
}
