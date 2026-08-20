import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

import { toast } from "sonner";
import { useApiMutation } from "./use-api-mutation";
import type { ApiResult } from "../api/client";

beforeEach(() => {
  vi.clearAllMocks();
});

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("useApiMutation", () => {
  it("toasts success and calls onSuccess when the underlying call resolves ok", async () => {
    const mutationFn = vi.fn(async (): Promise<ApiResult<{ id: string }>> => ({ ok: true, data: { id: "1" } }));
    const onSuccess = vi.fn();

    const { result } = renderHook(() => useApiMutation(mutationFn, { successMessage: "Identity created", onSuccess }), {
      wrapper,
    });

    result.current.mutate({});

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(toast.success).toHaveBeenCalledWith("Identity created");
    expect(onSuccess).toHaveBeenCalledWith({ id: "1" }, {});
    expect(toast.error).not.toHaveBeenCalled();
  });

  it("toasts the ApiError and does not call onSuccess when the underlying call fails", async () => {
    const mutationFn = vi.fn(async (): Promise<ApiResult<{ id: string }>> => ({
      ok: false,
      error: { kind: "validation", message: "email is required" },
    }));
    const onSuccess = vi.fn();

    const { result } = renderHook(() => useApiMutation(mutationFn, { successMessage: "should not show", onSuccess }), {
      wrapper,
    });

    result.current.mutate({});

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(toast.error).toHaveBeenCalledWith("email is required", expect.any(Object));
    expect(toast.success).not.toHaveBeenCalled();
    expect(onSuccess).not.toHaveBeenCalled();
  });

  it("supports a function successMessage computed from the result", async () => {
    const mutationFn = vi.fn(async (): Promise<ApiResult<{ name: string }>> => ({ ok: true, data: { name: "acme" } }));

    const { result } = renderHook(
      () => useApiMutation(mutationFn, { successMessage: (data) => `Created ${data.name}` }),
      { wrapper },
    );

    result.current.mutate({});

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(toast.success).toHaveBeenCalledWith("Created acme");
  });
});
