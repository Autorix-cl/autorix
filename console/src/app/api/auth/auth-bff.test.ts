import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

const cookieValues = new Map<string, string>();
vi.mock("next/headers", () => ({
  cookies: vi.fn(async () => ({
    get: (name: string) => {
      const value = cookieValues.get(name);
      return value !== undefined ? { value } : undefined;
    },
  })),
}));

import { clearSessionCache } from "@/lib/auth/session";
import { GET as me } from "./me/route";
import { POST as logout } from "./logout/route";
import { GET as samlLogin } from "./saml/login/route";
import { POST as samlCallback } from "./saml/callback/route";
import { GET as ssoCallback } from "./sso/callback/route";

describe("Auth BFF Security Routes", () => {
  beforeEach(() => {
    cookieValues.clear();
    clearSessionCache();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("GET /api/auth/me caches valid sessions briefly and invalidates immediately on logout", async () => {
    cookieValues.set("autorix_session", "sess-token-cache");
    const validResponse = () =>
      new Response(
        JSON.stringify({
          valid: true,
          operator: { id: "op-1", email: "admin@autorix.local", name: "Admin", role: "owner", is_active: true },
        }),
        { status: 200 },
      );
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(validResponse())
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ error: "expired" }), { status: 401 }));
    vi.stubGlobal("fetch", fetchMock);

    const [first, second] = await Promise.all([me(), me()]);
    expect(first.status).toBe(200);
    expect(second.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledTimes(1);

    const logoutRes = await logout();
    expect(logoutRes.status).toBe(200);

    const afterLogout = await me();
    expect(afterLogout.status).toBe(401);
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("GET /api/auth/me returns 503 without deleting session cookie on transient Argus 5xx errors", async () => {
    cookieValues.set("autorix_session", "sess-token-123");
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: "db busy" }), { status: 500 }));
    vi.stubGlobal("fetch", fetchMock);

    const res = await me();
    expect(res.status).toBe(503);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(res.headers.get("set-cookie")).toBeNull();
  });

  it("POST /api/auth/logout sets Clear-Site-Data header and deletes session cookie", async () => {
    cookieValues.set("autorix_session", "sess-token-123");
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 204 })));

    const res = await logout();
    expect(res.status).toBe(200);
    expect(res.headers.get("Clear-Site-Data")).toBe('"cookies", "storage", "cache"');
  });

  it("GET /api/auth/saml/login redirects to public Hermes /saml/login with RelayState", async () => {
    const prev = process.env.NEXT_PUBLIC_HERMES_URL;
    process.env.NEXT_PUBLIC_HERMES_URL = "https://hermes.public.example";
    try {
      const res = await samlLogin(new Request("https://console.example/api/auth/saml/login?provider=okta"));
      expect(res.status).toBe(307);
      const location = res.headers.get("location") || "";
      expect(location).toContain("https://hermes.public.example/saml/login");
      expect(location).not.toContain("/v1/saml/login");
      expect(location).toContain("provider=okta");
      expect(location).toContain("RelayState=");
    } finally {
      if (prev === undefined) delete process.env.NEXT_PUBLIC_HERMES_URL;
      else process.env.NEXT_PUBLIC_HERMES_URL = prev;
    }
  });

  it("POST /api/auth/saml/callback rejects missing or mismatched RelayState without minting a cookie", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);

    const form = new FormData();
    form.set("SAMLResponse", "arbitrary-unverified-blob");
    form.set("RelayState", "attacker-state");
    // No saved saml_relay_state cookie
    const res = await samlCallback(
      new Request("https://console.example/api/auth/saml/callback", { method: "POST", body: form }),
    );

    expect(res.status).toBe(307);
    expect(res.headers.get("location")).toContain("/login?error=saml_failed");
    expect(res.cookies.get("autorix_session")).toBeUndefined();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("POST /api/auth/saml/callback verifies assertion with Hermes /saml/acs and never mints fake ast_saml_ cookies on failure", async () => {
    cookieValues.set("saml_relay_state", "valid-relay-state");
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ error: "invalid signature" }), { status: 400 }));
    vi.stubGlobal("fetch", fetchMock);

    const form = new FormData();
    form.set("SAMLResponse", "bad-assertion");
    form.set("RelayState", "valid-relay-state");

    const res = await samlCallback(
      new Request("https://console.example/api/auth/saml/callback", { method: "POST", body: form }),
    );

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringMatching(/\/saml\/acs$/),
      expect.objectContaining({ method: "POST" }),
    );
    expect(res.status).toBe(307);
    expect(res.headers.get("location")).toContain("/login?error=saml_failed");
    expect(res.cookies.get("autorix_session")).toBeUndefined();
  });

  it("GET /api/auth/sso/callback verifies token via Janus userinfo and never mints fake ast_sso_ cookies when Argus rejects", async () => {
    cookieValues.set("sso_state", "valid-state");
    cookieValues.set("sso_verifier", "valid-verifier");

    const fetchMock = vi.fn().mockImplementation(async (url: RequestInfo | URL) => {
      const u = String(url);
      if (u.includes("/.well-known/openid-configuration")) {
        return new Response(
          JSON.stringify({
            issuer: "http://janus:4444",
            authorization_endpoint: "http://localhost:4444/oauth2/auth",
            token_endpoint: "http://janus:4444/oauth2/token",
            userinfo_endpoint: "http://janus:4444/oauth2/userinfo",
            jwks_uri: "http://janus:4444/.well-known/jwks.json",
            response_types_supported: ["code"],
            subject_types_supported: ["public"],
            id_token_signing_alg_values_supported: ["RS256"],
          }),
          { status: 200 },
        );
      }
      if (u.includes("/oauth2/token")) {
        return new Response(JSON.stringify({ access_token: "janus-access-token" }), { status: 200 });
      }
      if (u.includes("/oauth2/userinfo")) {
        return new Response(JSON.stringify({ sub: "u1", email: "alice@autorix.io", name: "Alice" }), { status: 200 });
      }
      if (u.includes("/v1/auth/login")) {
        return new Response(JSON.stringify({ error: "operator not provisioned" }), { status: 401 });
      }
      return new Response("Not Found", { status: 404 });
    });
    vi.stubGlobal("fetch", fetchMock);

    const res = await ssoCallback(
      new Request("https://console.example/api/auth/sso/callback?code=auth-code&state=valid-state"),
    );

    expect(res.status).toBe(307);
    expect(res.headers.get("location")).toContain("/login?error=sso_failed");
    expect(res.cookies.get("autorix_session")).toBeUndefined();
    expect(fetchMock.mock.calls.some(([u]) => String(u).includes("/oauth2/userinfo"))).toBe(true);
  });
});
