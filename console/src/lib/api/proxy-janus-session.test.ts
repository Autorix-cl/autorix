import { afterEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";
import { proxyRequest } from "./proxy";

vi.mock("next/headers", () => ({
  cookies: vi.fn(async () => ({
    get: (name: string) => (name === "autorix_session" ? { value: "fabricated-session" } : undefined),
  })),
}));

afterEach(() => vi.unstubAllGlobals());

describe("Private engine BFF session boundary", () => {
  it.each([
    ["janus", "/admin/clients"],
    ["aegis", "/rules"],
  ] as const)("rejects a fabricated cookie before forwarding to %s", async (service, path) => {
    const fetchMock = vi.fn().mockResolvedValue(new Response("Unauthorized", { status: 401 }));
    vi.stubGlobal("fetch", fetchMock);

    const response = await proxyRequest(service, path, z.unknown(), { method: "POST" });

    expect(response.status).toBe(401);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0][0]).toMatch(/\/v1\/auth\/session$/);
    expect(fetchMock.mock.calls[0][1].headers.Authorization).toBe("Bearer fabricated-session");
  });
});
