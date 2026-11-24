import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";

const mockUsePathname = vi.fn(() => "/identities");
vi.mock("next/navigation", () => ({
  usePathname: () => mockUsePathname(),
}));

import { AuthProvider, usePermission } from "./use-permission";
import { setRedirectHandler, resetRedirectFlag } from "../api/client";

function wrapper({ children }: { children: ReactNode }) {
  return <AuthProvider>{children}</AuthProvider>;
}

beforeEach(() => {
  mockUsePathname.mockReturnValue("/identities");
});

afterEach(() => {
  vi.unstubAllGlobals();
  resetRedirectFlag();
});

describe("AuthProvider", () => {
  it("populates operator and permissions when /api/auth/me returns authenticated session", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            authenticated: true,
            operator: {
              id: "op-1",
              name: "Admin User",
              email: "admin@autorix.io",
              role: "admin",
              is_local: true,
              is_active: true,
            },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } }
        )
      )
    );

    const redirectSpy = vi.fn();
    setRedirectHandler(redirectSpy);

    const { result } = renderHook(() => usePermission(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.operator?.email).toBe("admin@autorix.io");
    expect(result.current.can("policies:write")).toBe(true);
    expect(redirectSpy).not.toHaveBeenCalled();
  });

  it("calls redirectToLogin when /api/auth/me returns 401 unauthenticated session", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ authenticated: false }), {
          status: 401,
          headers: { "Content-Type": "application/json" },
        })
      )
    );

    const redirectSpy = vi.fn();
    setRedirectHandler(redirectSpy);

    const { result } = renderHook(() => usePermission(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.operator).toBeNull();
    expect(redirectSpy).toHaveBeenCalledTimes(1);
    expect(redirectSpy).toHaveBeenCalledWith(expect.stringContaining("/login?from="));
  });

  it("does not refetch /api/auth/me on navigation between protected routes, but refetches after leaving /login", async () => {
    const fetchMock = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(
          JSON.stringify({
            authenticated: true,
            operator: {
              id: "op-1",
              name: "Admin User",
              email: "admin@autorix.io",
              role: "admin",
              is_local: true,
              is_active: true,
            },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    mockUsePathname.mockReturnValue("/identities");
    const { result, rerender } = renderHook(() => usePermission(), { wrapper });
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(fetchMock).toHaveBeenCalledTimes(1);

    mockUsePathname.mockReturnValue("/policies");
    rerender();
    expect(fetchMock).toHaveBeenCalledTimes(1);

    mockUsePathname.mockReturnValue("/login");
    rerender();
    expect(fetchMock).toHaveBeenCalledTimes(1);

    mockUsePathname.mockReturnValue("/identities");
    rerender();
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
  });
});
