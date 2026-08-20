import { describe, it, expect } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useApiQuery } from "./use-api-query";
import type { ApiResult } from "../api/client";

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("useApiQuery", () => {
  it("resolves with data when the underlying call succeeds", async () => {
    const queryFn = async (): Promise<ApiResult<{ id: string }[]>> => ({ ok: true, data: [{ id: "1" }] });

    const { result } = renderHook(() => useApiQuery(["identities"], queryFn), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([{ id: "1" }]);
  });

  it("surfaces the ApiError (with its kind) as the query's error, not a generic Error", async () => {
    const queryFn = async (): Promise<ApiResult<{ id: string }[]>> => ({
      ok: false,
      error: { kind: "engine-unreachable", message: "network error" },
    });

    const { result } = renderHook(() => useApiQuery(["identities"], queryFn), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.kind).toBe("engine-unreachable");
    expect(result.current.error?.message).toBe("network error");
  });

  it("exposes isLoading while the query is in flight", () => {
    const queryFn = () => new Promise<ApiResult<{ id: string }[]>>(() => {}); // never resolves

    const { result } = renderHook(() => useApiQuery(["identities"], queryFn), { wrapper });

    expect(result.current.isLoading).toBe(true);
  });
});
