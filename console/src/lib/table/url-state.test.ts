import { describe, it, expect } from "vitest";
import { serializeTableParams, deserializeTableParams } from "./url-state";

describe("Table URL-encoded state (P4-S3-T2)", () => {
  it("serializes table pagination, sort and search into URL search params", () => {
    const params = serializeTableParams({
      page: 2,
      pageSize: 25,
      sortBy: "email",
      sortOrder: "asc",
      search: "alice",
      filters: { status: "active" },
    });

    expect(params.get("page")).toBe("2");
    expect(params.get("pageSize")).toBe("25");
    expect(params.get("sortBy")).toBe("email");
    expect(params.get("sortOrder")).toBe("asc");
    expect(params.get("q")).toBe("alice");
    expect(params.get("f_status")).toBe("active");
  });

  it("deserializes URL search params back into structured table query state", () => {
    const searchParams = new URLSearchParams("page=3&pageSize=50&sortBy=created_at&sortOrder=desc&q=bob&f_status=suspended");
    const state = deserializeTableParams(searchParams);

    expect(state.page).toBe(3);
    expect(state.pageSize).toBe(50);
    expect(state.sortBy).toBe("created_at");
    expect(state.sortOrder).toBe("desc");
    expect(state.search).toBe("bob");
    expect(state.filters).toEqual({ status: "suspended" });
  });

  it("falls back to sensible defaults on empty search params", () => {
    const searchParams = new URLSearchParams("");
    const state = deserializeTableParams(searchParams);

    expect(state.page).toBe(1);
    expect(state.pageSize).toBe(10);
    expect(state.sortBy).toBeUndefined();
    expect(state.sortOrder).toBe("asc");
    expect(state.search).toBe("");
    expect(state.filters).toEqual({});
  });
});
