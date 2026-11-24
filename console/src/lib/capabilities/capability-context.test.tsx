import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import * as React from "react";

const mockUsePathname = vi.fn(() => "/");
vi.mock("next/navigation", () => ({
  usePathname: () => mockUsePathname(),
}));

import { EnvironmentProvider, useEnvironment } from "@/lib/environment/environment-context";
import { CapabilityProvider, useCapabilities } from "./capability-context";

describe("CapabilityContext", () => {
  beforeEach(() => {
    localStorage.clear();
    mockUsePathname.mockReturnValue("/");
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify([]), { status: 200 }))),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("provides healthy engines in production by default", () => {
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <EnvironmentProvider>
        <CapabilityProvider>{children}</CapabilityProvider>
      </EnvironmentProvider>
    );

    const { result } = renderHook(() => useCapabilities(), { wrapper });
    expect(result.current.isEngineConnected("nexus")).toBe(true);
    expect(result.current.getEngineStatus("nexus")).toBe("healthy");
  });

  it("marks engines as not_connected when switched to staging without instances", async () => {
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <EnvironmentProvider>
        <CapabilityProvider>{children}</CapabilityProvider>
      </EnvironmentProvider>
    );

    const { result } = renderHook(
      () => ({
        env: useEnvironment(),
        caps: useCapabilities(),
      }),
      { wrapper },
    );

    await act(async () => {
      result.current.env.setEnvironment("staging");
    });

    expect(result.current.caps.isEngineConnected("nexus")).toBe(false);
    expect(result.current.caps.getEngineStatus("nexus")).toBe("not_connected");
  });

  it("sets explicit error and marks engines unreachable when fleet API fails instead of fabricating healthy state", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation((url: RequestInfo | URL) => {
        if (String(url).includes("/api/fleet/engines")) {
          return Promise.resolve(new Response(JSON.stringify({ error: "argus offline" }), { status: 502 }));
        }
        return Promise.resolve(new Response(JSON.stringify([]), { status: 200 }));
      }),
    );

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <EnvironmentProvider>
        <CapabilityProvider>{children}</CapabilityProvider>
      </EnvironmentProvider>
    );

    const { result } = renderHook(() => useCapabilities(), { wrapper });

    await waitFor(() => expect(result.current.error).toBeTruthy());
    expect(result.current.getEngineStatus("nexus")).toBe("unreachable");
    expect(result.current.isEngineConnected("nexus")).toBe(false);
  });

  it("does NOT fetch /api/fleet/engines on public auth routes", async () => {
    mockUsePathname.mockReturnValue("/login");
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify([]), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <EnvironmentProvider>
        <CapabilityProvider>{children}</CapabilityProvider>
      </EnvironmentProvider>
    );

    renderHook(() => useCapabilities(), { wrapper });
    await act(async () => {
      await Promise.resolve();
    });

    expect(fetchMock).not.toHaveBeenCalledWith(expect.stringContaining("/api/fleet/engines"));
  });
});
