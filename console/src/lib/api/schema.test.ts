import { describe, it, expect, vi, afterEach } from "vitest";
import { z } from "zod";
import { parseWithSchema, fetchAndParse } from "./schema";

afterEach(() => {
  vi.unstubAllGlobals();
});

const widgetSchema = z.object({ id: z.string(), count: z.number() });

describe("parseWithSchema", () => {
  it("returns ok:true with the parsed value when data matches the schema", () => {
    const result = parseWithSchema(widgetSchema, { id: "w1", count: 3 });
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data).toEqual({ id: "w1", count: 3 });
    }
  });

  it("returns a validation ApiError when data does not match the schema", () => {
    const result = parseWithSchema(widgetSchema, { id: "w1", count: "not-a-number" });
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.kind).toBe("validation");
      expect(result.error.message).toContain("count");
    }
  });

  it("rejects a response shape that changed out from under the client (e.g. a missing field)", () => {
    const result = parseWithSchema(widgetSchema, { id: "w1" });
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.message).toContain("count");
    }
  });
});

describe("fetchAndParse", () => {
  it("parses the response body against schema on success", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: "w1", count: 3 }), { status: 200 })),
    );

    const result = await fetchAndParse("/api/widgets/w1", widgetSchema);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data.id).toBe("w1");
    }
  });

  it("short-circuits with the transport error without attempting to parse when the fetch itself fails", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("fetch failed")));

    const result = await fetchAndParse("/api/widgets/w1", widgetSchema);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.kind).toBe("engine-unreachable");
    }
  });

  it("surfaces a validation error when the response is 200 but doesn't match the schema", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: "w1" }), { status: 200 })));

    const result = await fetchAndParse("/api/widgets/w1", widgetSchema);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.kind).toBe("validation");
    }
  });
});
