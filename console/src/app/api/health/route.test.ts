import { describe, it, expect, vi, afterEach } from "vitest";
import type { ServiceHealth } from "@/lib/api/schemas/health";

vi.mock("@/lib/api-config", () => ({
  getServiceUrl: (service: string) => `http://${service}:9999`,
}));

afterEach(() => {
  vi.unstubAllGlobals();
  vi.resetModules();
});

async function loadRoute() {
  // Fresh import per test so module-level state (none expected, but future-proof) doesn't leak.
  return await import("./route");
}

describe("GET /api/health", () => {
  it("reports healthy for a service whose /health/ready returns 200", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ status: "ready" }), { status: 200 })),
    );

    const { GET } = await loadRoute();
    const res = await GET();
    const body = await res.json();

    expect(body.services.every((s: ServiceHealth) => s.status === "healthy")).toBe(true);
    expect(body.services.every((s: ServiceHealth) => typeof s.latencyMs === "number")).toBe(true);
  });

  it("reports degraded for a service whose /health/ready returns 503", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ status: "not_ready" }), { status: 503 })),
    );

    const { GET } = await loadRoute();
    const res = await GET();
    const body = await res.json();

    expect(body.services.every((s: ServiceHealth) => s.status === "degraded")).toBe(true);
  });

  it("reports unreachable (never healthy) for a service whose fetch throws or times out", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("fetch failed")));

    const { GET } = await loadRoute();
    const res = await GET();
    const body = await res.json();

    expect(body.services.every((s: ServiceHealth) => s.status === "unreachable")).toBe(true);
    expect(body.services.every((s: ServiceHealth) => s.latencyMs === null)).toBe(true);
  });

  it("never returns online:true / healthy for every service unconditionally (no simulation fallback)", async () => {
    // Half the engines respond, half fail — the old code's catch-block used to paper over
    // any failure with a hardcoded "healthy". This proves that's gone.
    let call = 0;
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(() => {
        call += 1;
        if (call % 2 === 0) return Promise.reject(new TypeError("fetch failed"));
        return Promise.resolve(new Response(JSON.stringify({ status: "ready" }), { status: 200 }));
      }),
    );

    const { GET } = await loadRoute();
    const res = await GET();
    const body = await res.json();

    const statuses = new Set(body.services.map((s: ServiceHealth) => s.status));
    expect(statuses.has("unreachable")).toBe(true);
    expect(statuses.has("healthy")).toBe(true);
  });

  it("includes all 7 engines, including nexus and aegis via their real REST admin health endpoint", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ status: "ready" }), { status: 200 })),
    );

    const { GET } = await loadRoute();
    const res = await GET();
    const body = await res.json();

    const keys = body.services.map((s: ServiceHealth) => s.key).sort();
    expect(keys).toEqual(["aegis", "ego", "hermes", "janus", "nexus", "themis", "vulcan"]);
  });

  it("probes /health/ready specifically, not a business endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ status: "ready" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    const { GET } = await loadRoute();
    await GET();

    for (const [url] of fetchMock.mock.calls) {
      expect(String(url)).toMatch(/\/health\/ready$/);
    }
  });

  it("returns a top-level checkedAt timestamp", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ status: "ready" }), { status: 200 })),
    );

    const { GET } = await loadRoute();
    const res = await GET();
    const body = await res.json();

    expect(() => new Date(body.checkedAt).toISOString()).not.toThrow();
  });

  it("does not include a fabricated onlineCount/totalCount computed server-side", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ status: "ready" }), { status: 200 })),
    );

    const { GET } = await loadRoute();
    const res = await GET();
    const body = await res.json();

    expect(body.onlineCount).toBeUndefined();
    expect(body.totalCount).toBeUndefined();
  });
});
