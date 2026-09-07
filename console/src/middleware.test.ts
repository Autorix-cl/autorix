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
});
