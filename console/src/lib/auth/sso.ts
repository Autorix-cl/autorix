import { getServiceUrl } from "../api-config";

export interface OidcDiscovery {
  issuer: string;
  authorization_endpoint: string;
  token_endpoint: string;
  jwks_uri: string;
  response_types_supported: string[];
  subject_types_supported: string[];
  id_token_signing_alg_values_supported: string[];
}

export async function getJanusDiscovery(): Promise<OidcDiscovery | null> {
  try {
    const janusUrl = getServiceUrl("janus");
    const res = await fetch(`${janusUrl}/oauth2/openid-configuration`, {
      cache: "no-store",
    });
    if (!res.ok) {
      // Fallback standard Janus discovery paths
      return {
        issuer: janusUrl,
        authorization_endpoint: `${janusUrl}/oauth2/auth`,
        token_endpoint: `${janusUrl}/oauth2/token`,
        jwks_uri: `${janusUrl}/oauth2/jwks`,
        response_types_supported: ["code"],
        subject_types_supported: ["public"],
        id_token_signing_alg_values_supported: ["RS256", "ES256"],
      };
    }
    return await res.json();
  } catch {
    const janusUrl = getServiceUrl("janus");
    return {
      issuer: janusUrl,
      authorization_endpoint: `${janusUrl}/oauth2/auth`,
      token_endpoint: `${janusUrl}/oauth2/token`,
      jwks_uri: `${janusUrl}/oauth2/jwks`,
      response_types_supported: ["code"],
      subject_types_supported: ["public"],
      id_token_signing_alg_values_supported: ["RS256", "ES256"],
    };
  }
}

export function generatePkceVerifier(): string {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

export async function generatePkceChallenge(verifier: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(verifier);
  const hash = await crypto.subtle.digest("SHA-256", data);
  const bytes = new Uint8Array(hash);
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

/**
 * Self-registers console client in Janus if missing (P3-S3-T2).
 */
export async function registerConsoleOAuthClient(): Promise<void> {
  try {
    const janusUrl = getServiceUrl("janus");
    await fetch(`${janusUrl}/oauth2/clients`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        client_id: "autorix-console",
        client_name: "Autorix Control Plane Console",
        redirect_uris: ["/api/auth/sso/callback"],
        grant_types: ["authorization_code", "refresh_token"],
        response_types: ["code"],
        token_endpoint_auth_method: "none", // Public PKCE client
      }),
    });
  } catch {
    // Janus will accept the client or use pre-seeded client ID
  }
}
