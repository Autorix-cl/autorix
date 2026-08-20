import { NextResponse } from "next/server";
import { getJanusDiscovery, generatePkceVerifier, generatePkceChallenge, registerConsoleOAuthClient } from "@/lib/auth/sso";

export async function GET(request: Request) {
  try {
    await registerConsoleOAuthClient();

    const discovery = await getJanusDiscovery();
    if (!discovery) {
      return NextResponse.json(
        { error: "Janus OIDC provider discovery unavailable" },
        { status: 502 }
      );
    }

    const state = crypto.randomUUID();
    const verifier = generatePkceVerifier();
    const challenge = await generatePkceChallenge(verifier);

    const redirectUri = new URL("/api/auth/sso/callback", request.url).toString();
    const authUrl = new URL(discovery.authorization_endpoint);
    authUrl.searchParams.set("response_type", "code");
    authUrl.searchParams.set("client_id", "autorix-console");
    authUrl.searchParams.set("redirect_uri", redirectUri);
    authUrl.searchParams.set("scope", "openid profile email roles");
    authUrl.searchParams.set("state", state);
    authUrl.searchParams.set("code_challenge", challenge);
    authUrl.searchParams.set("code_challenge_method", "S256");

    const response = NextResponse.redirect(authUrl.toString());
    response.cookies.set("sso_state", state, { httpOnly: true, secure: process.env.NODE_ENV === "production", sameSite: "lax", maxAge: 600 });
    response.cookies.set("sso_verifier", verifier, { httpOnly: true, secure: process.env.NODE_ENV === "production", sameSite: "lax", maxAge: 600 });

    return response;
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : "Internal server error";
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
