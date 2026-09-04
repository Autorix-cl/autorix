import { NextResponse } from "next/server";
import { getCurrentOperator, SESSION_COOKIE_NAME } from "@/lib/auth/session";

export async function GET() {
  try {
    const operator = await getCurrentOperator();
    if (!operator) {
      const response = NextResponse.json({ authenticated: false }, { status: 401 });
      response.cookies.delete(SESSION_COOKIE_NAME);
      return response;
    }
    return NextResponse.json({
      authenticated: true,
      operator,
    });
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : "Internal server error";
    return NextResponse.json(
      { error: message },
      { status: 500 }
    );
  }
}
