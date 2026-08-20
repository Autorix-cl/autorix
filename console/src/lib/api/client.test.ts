import { describe, it, expect, vi, afterEach } from "vitest";
import { fetchJSON } from "./client";

function jsonResponse(body: unknown, init: { status?: number; headers?: Record<string, string> } = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: { "Content-Type": "application/json", ...(init.headers ?? {}) },
  });
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("fetchJSON", () => {
  it("returns ok:true with parsed data on a 200 response", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ id: "1", name: "alice" })));

    const result = await fetchJSON<{ id: string; name: string }>("/api/identities");

    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data).toEqual({ id: "1", name: "alice" });
    }
  });

  it("classifies a 401 as unauthorized", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ error: "no session" }, { status: 401 })));

    const result = await fetchJSON("/api/identities");

    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.kind).toBe("unauthorized");
      expect(result.error.status).toBe(401);
      expect(result.error.message).toBe("no session");
    }
  });

  it("classifies a 422 as validation", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ error: "email is required" }, { status: 422 })));

    const result = await fetchJSON("/api/identities");

    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.kind).toBe("validation");
    }
  });

  it("classifies a 500 as engine-error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ error: "internal" }, { status: 500 })));

    const result = await fetchJSON("/api/identities");

    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.kind).toBe("engine-error");
    }
  });

  it("classifies a network failure (fetch throwing) as engine-unreachable", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("fetch failed")));

    const result = await fetchJSON("/api/identities");

    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.kind).toBe("engine-unreachable");
    }
  });

  it("classifies a timeout as engine-unreachable, not engine-error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation((_url: string, init?: RequestInit) => {
        return new Promise((_resolve, reject) => {
          init?.signal?.addEventListener("abort", () => {
            reject(new DOMException("aborted", "AbortError"));
          });
        });
      }),
    );

    const result = await fetchJSON("/api/identities", { timeoutMs: 5 });

    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.kind).toBe("engine-unreachable");
      expect(result.error.message).toMatch(/timed out/i);
    }
  });

  it("propagates the upstream x-request-id header on both success and error", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(jsonResponse({ error: "boom" }, { status: 500, headers: { "x-request-id": "req-42" } })),
    );

    const result = await fetchJSON("/api/identities");

    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.requestId).toBe("req-42");
    }
  });

  it("falls back to raw text when the error body is not JSON", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          new Response("upstream is on fire", { status: 502, headers: { "Content-Type": "text/plain" } }),
        ),
    );

    const result = await fetchJSON("/api/identities");

    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.kind).toBe("engine-error");
      expect(result.error.message).toContain("upstream is on fire");
    }
  });

  it("classifies an unmapped 4xx status as unknown", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ error: "teapot" }, { status: 418 })));

    const result = await fetchJSON("/api/identities");

    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.kind).toBe("unknown");
    }
  });
});
