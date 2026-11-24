import { BACKEND_URLS, getServiceUrl, JANUS_ADMIN_URL } from "../api-config";
import { buildInternalAuthHeaders } from "../api/internal-headers";

const SSO_REQUEST_TIMEOUT_MS = 8_000;

export interface OidcDiscovery {
  issuer: string;
  authorization_endpoint: string;
  token_endpoint: string;
  userinfo_endpoint?: string;
  introspection_endpoint?: string;
  jwks_uri: string;
  response_types_supported: string[];
  subject_types_supported: string[];
  id_token_signing_alg_values_supported: string[];
}

export function getPublicJanusUrl(): string {
  return (process.env.NEXT_PUBLIC_JANUS_URL || "http://localhost:4444").replace(/\/+$/, "");
}

function buildFallbackDiscovery(): OidcDiscovery {
  const internalJanusUrl = getServiceUrl("janus").replace(/\/+$/, "");
  const publicJanusUrl = getPublicJanusUrl();
  return {
    issuer: publicJanusUrl,
    authorization_endpoint: `${publicJanusUrl}/oauth2/auth`,
    token_endpoint: `${internalJanusUrl}/oauth2/token`,
    userinfo_endpoint: `${internalJanusUrl}/oauth2/userinfo`,
    introspection_endpoint: `${internalJanusUrl}/oauth2/introspect`,
    jwks_uri: `${internalJanusUrl}/.well-known/jwks.json`,
    response_types_supported: ["code"],
    subject_types_supported: ["public"],
    id_token_signing_alg_values_supported: ["RS256", "ES256"],
  };
}

function toBrowserAuthorizationEndpoint(endpoint: string, internalJanusUrl: string, publicJanusUrl: string): string {
  const backendJanusUrl = BACKEND_URLS.janus.replace(/\/+$/, "");
  for (const internalOrigin of [internalJanusUrl, backendJanusUrl]) {
    if (internalOrigin && endpoint.startsWith(internalOrigin)) {
      return `${publicJanusUrl}${endpoint.slice(internalOrigin.length)}`;
    }
  }
  return endpoint;
}

export async function getJanusDiscovery(): Promise<OidcDiscovery | null> {
  const internalJanusUrl = getServiceUrl("janus").replace(/\/+$/, "");
  const publicJanusUrl = getPublicJanusUrl();

  try {
    const res = await fetch(`${internalJanusUrl}/.well-known/openid-configuration`, {
      cache: "no-store",
      signal: AbortSignal.timeout(SSO_REQUEST_TIMEOUT_MS),
    });
    if (!res.ok) {
      return buildFallbackDiscovery();
    }
    const discovery = (await res.json()) as OidcDiscovery;
    return {
      ...discovery,
      authorization_endpoint: toBrowserAuthorizationEndpoint(
        discovery.authorization_endpoint || `${publicJanusUrl}/oauth2/auth`,
        internalJanusUrl,
        publicJanusUrl,
      ),
      token_endpoint: discovery.token_endpoint || `${internalJanusUrl}/oauth2/token`,
      userinfo_endpoint: discovery.userinfo_endpoint || `${internalJanusUrl}/oauth2/userinfo`,
      introspection_endpoint: discovery.introspection_endpoint || `${internalJanusUrl}/oauth2/introspect`,
      jwks_uri: discovery.jwks_uri || `${internalJanusUrl}/.well-known/jwks.json`,
    };
  } catch {
    return buildFallbackDiscovery();
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
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

/**
 * Self-registers console client in Janus over the private admin listener if missing (P3-S3-T2).
 */
export async function registerConsoleOAuthClient(): Promise<void> {
  try {
    const adminBase = JANUS_ADMIN_URL.replace(/\/+$/, "");
    await fetch(`${adminBase}/admin/clients`, {
      method: "POST",
      headers: buildInternalAuthHeaders({ "Content-Type": "application/json" }),
      signal: AbortSignal.timeout(SSO_REQUEST_TIMEOUT_MS),
      body: JSON.stringify({
        client_id: "autorix-console",
        client_name: "Autorix Control Plane Console",
        redirect_uris: ["/api/auth/sso/callback"],
        grant_types: ["authorization_code", "refresh_token"],
        response_types: ["code"],
        token_endpoint_auth_method: "none",
      }),
    });
  } catch {
    // Janus may already have the pre-seeded client ID
  }
}
