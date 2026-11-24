import { NextResponse } from "next/server";

function getPublicHermesUrl(): string {
  return (process.env.NEXT_PUBLIC_HERMES_URL || "http://localhost:4477").replace(/\/+$/, "");
}

export async function GET(request: Request) {
  try {
    const hermesPublicUrl = getPublicHermesUrl();
    const relayState = crypto.randomUUID();
    const samlUrl = new URL(`${hermesPublicUrl}/saml/login`);
    samlUrl.searchParams.set("RelayState", relayState);

    const { searchParams } = new URL(request.url);
    const provider = searchParams.get("provider");
    if (provider) {
      samlUrl.searchParams.set("provider", provider);
    }

    const response = NextResponse.redirect(samlUrl.toString());
    response.cookies.set("saml_relay_state", relayState, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: 600,
    });
    return response;
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : "Internal server error";
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
