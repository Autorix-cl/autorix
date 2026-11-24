import { NextRequest } from "next/server";
import { describe, expect, it } from "vitest";

import { middleware } from "./middleware";

describe("CSRF origin verification", () => {
  it("rejects an attacker origin that merely contains the trusted host", async () => {
    const request = new NextRequest("https://console.example/api/oauth2/clients", {
      method: "POST",
      headers: { Origin: "https://console.example.attacker.test" },
    });
    request.cookies.set("autorix_session", "session");

    const response = await middleware(request);
    expect(response.status).toBe(403);
  });

  it("accepts the exact origin for a protected mutation", async () => {
    const request = new NextRequest("https://console.example/api/oauth2/clients", {
      method: "POST",
      headers: { Origin: "https://console.example" },
    });
    request.cookies.set("autorix_session", "session");

    const response = await middleware(request);
    expect(response.status).toBe(200);
  });

  it("accepts matching double-submit CSRF token on a protected mutation", async () => {
    const request = new NextRequest("https://console.example/api/oauth2/clients", {
      method: "POST",
      headers: { "X-CSRF-Token": "abc123token" },
    });
    request.cookies.set("autorix_session", "session");
    request.cookies.set("autorix_csrf", "abc123token");

    const response = await middleware(request);
    expect(response.status).toBe(200);
  });
});

describe("API dot-path bypass prevention and auth allowlist", () => {
  it.each(["/api/identities/alice@example.com", "/api/oauth2/scopes/profile.read", "/api/files/export.svg"])(
    "rejects unauthenticated GET %s with 401 even when path contains a dot",
    async (path) => {
      const request = new NextRequest(`https://console.example${path}`, { method: "GET" });
      const response = await middleware(request);
      expect(response.status).toBe(401);
    },
  );

  it.each(["/api/identities/alice@example.com", "/api/oauth2/scopes/profile.read"])(
    "enforces CSRF on cross-origin POST %s with 403 even when path contains a dot",
    async (path) => {
      const request = new NextRequest(`https://console.example${path}`, {
        method: "POST",
        headers: { Origin: "https://attacker.example" },
      });
      request.cookies.set("autorix_session", "session");

      const response = await middleware(request);
      expect(response.status).toBe(403);
    },
  );

  it("allows genuine non-API static assets without a session", async () => {
    const request = new NextRequest("https://console.example/logo.svg", { method: "GET" });
    const response = await middleware(request);
    expect(response.status).toBe(200);
  });

  it("enforces session and CSRF protection on POST /api/auth/logout", async () => {
    const unauth = new NextRequest("https://console.example/api/auth/logout", { method: "POST" });
    expect((await middleware(unauth)).status).toBe(401);

    const crossOrigin = new NextRequest("https://console.example/api/auth/logout", {
      method: "POST",
      headers: { Origin: "https://attacker.example" },
    });
    crossOrigin.cookies.set("autorix_session", "session");
    expect((await middleware(crossOrigin)).status).toBe(403);

    const sameOrigin = new NextRequest("https://console.example/api/auth/logout", {
      method: "POST",
      headers: { Origin: "https://console.example" },
    });
    sameOrigin.cookies.set("autorix_session", "session");
    expect((await middleware(sameOrigin)).status).toBe(200);
  });

  it.each([
    "/api/auth/sso/login",
    "/api/auth/sso/callback",
    "/api/auth/saml/login",
    "/api/auth/saml/callback",
    "/api/health",
  ])("allows unauthenticated access to public route %s", async (path) => {
    const request = new NextRequest(`https://console.example${path}`, { method: "GET" });
    const response = await middleware(request);
    expect(response.status).toBe(200);
  });
});
