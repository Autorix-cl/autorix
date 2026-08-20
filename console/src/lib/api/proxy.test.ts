import { describe, it, expect, vi, afterEach } from "vitest";
import { z } from "zod";
import { proxyRequest } from "./proxy";

afterEach(() => {
  vi.unstubAllGlobals();
});

const widgetSchema = z.object({ id: z.string(), count: z.number() });

describe("proxyRequest", () => {
  it("returns the upstream body with a 200 and an x-request-id header on success", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: "w1", count: 3 }), { status: 200 })),
    );

    const res = await proxyRequest("ego", "/widgets", widgetSchema);

    expect(res.status).toBe(200);
    expect(res.headers.get("x-request-id")).toBeTruthy();
    await expect(res.json()).resolves.toEqual({ id: "w1", count: 3 });
  });

  it("propagates the upstream status and error message on a 4xx/5xx instead of swallowing it into 200", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: "duplicate id" }), { status: 409 })),
    );

    const res = await proxyRequest("ego", "/widgets", widgetSchema);

    expect(res.status).toBe(409);
    const body = await res.json();
    expect(body.error).toBe("duplicate id");
  });

  it("returns 502 with a network-failure message when the upstream is unreachable, never a silent empty success", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("fetch failed")));

    const res = await proxyRequest("ego", "/widgets", widgetSchema);

    expect(res.status).toBe(502);
    const body = await res.json();
    expect(body.error).toBeTruthy();
  });

  it("returns 502 when the upstream response doesn't match the declared schema", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: "w1" }), { status: 200 })));

    const res = await proxyRequest("ego", "/widgets", widgetSchema);

    expect(res.status).toBe(502);
    const body = await res.json();
    expect(body.error).toMatch(/unexpected response shape/i);
  });

  it("forwards method, headers and body from the init argument", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: "w1", count: 1 }), { status: 201 }));
    vi.stubGlobal("fetch", fetchMock);

    await proxyRequest("vulcan", "/keys", widgetSchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: "test" }),
    });

    const [, init] = fetchMock.mock.calls[0];
    expect(init.method).toBe("POST");
    expect(init.body).toBe(JSON.stringify({ name: "test" }));
  });

  it("preserves the upstream status on a successful non-200 (e.g. 201 Created)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: "w1", count: 1 }), { status: 201 })),
    );

    const res = await proxyRequest("vulcan", "/keys", widgetSchema);

    expect(res.status).toBe(201);
  });

  it("falls back to raw text for the error message when the error body is not JSON", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("upstream is on fire", { status: 502 })));

    const res = await proxyRequest("ego", "/widgets", widgetSchema);

    expect(res.status).toBe(502);
    const body = await res.json();
    expect(body.error).toContain("upstream is on fire");
  });

  it("rejects unknown backend services with 403 SSRF protection", async () => {
    const res = await proxyRequest("unregistered-service" as unknown as "ego", "/test", widgetSchema);
    expect(res.status).toBe(403);
    const body = await res.json();
    expect(body.error).toContain("is not a valid registered engine");
  });
});
