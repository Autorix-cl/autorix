import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import * as React from "react";
import { EnvironmentProvider, useEnvironment } from "@/lib/environment/environment-context";
import { CapabilityProvider, useCapabilities } from "./capability-context";

describe("CapabilityContext", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify([]), { status: 200 })));
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
});
