import { NextResponse } from "next/server";
import { getServiceUrl } from "@/lib/api-config";

export async function GET() {
  try {
    const hermesUrl = getServiceUrl("hermes");
    const relayState = crypto.randomUUID();
    const samlUrl = new URL(`${hermesUrl}/v1/saml/login`);
    samlUrl.searchParams.set("RelayState", relayState);

    const response = NextResponse.redirect(samlUrl.toString());
    response.cookies.set("saml_relay_state", relayState, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      maxAge: 600,
    });
    return response;
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : "Internal server error";
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
