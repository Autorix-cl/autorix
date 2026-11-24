import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import * as React from "react";

const mockUsePathname = vi.fn(() => "/");
vi.mock("next/navigation", () => ({
  usePathname: () => mockUsePathname(),
}));

import { EnvironmentProvider, useEnvironment } from "./environment-context";

describe("EnvironmentContext", () => {
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
    vi.restoreAllMocks();
  });

  it("defaults to Production environment", () => {
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <EnvironmentProvider>{children}</EnvironmentProvider>
    );
    const { result } = renderHook(() => useEnvironment(), { wrapper });

    expect(result.current.currentEnv.id).toBe("prod");
    expect(result.current.currentEnv.name).toBe("Production");
    expect(result.current.isProduction).toBe(true);
  });

  it("switches environment and persists to localStorage", () => {
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <EnvironmentProvider>{children}</EnvironmentProvider>
    );
    const { result } = renderHook(() => useEnvironment(), { wrapper });

    act(() => {
      result.current.setEnvironment("staging");
    });

    expect(result.current.currentEnv.id).toBe("staging");
    expect(result.current.currentEnv.name).toBe("Staging");
    expect(result.current.isProduction).toBe(false);
    expect(localStorage.getItem("autorix_active_env")).toBe("staging");
  });

  it("loads remote environments from /api/governance/environments on authenticated routes", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify([
            { id: "prod", slug: "production", name: "Production EU", is_production: true },
            { id: "qa", slug: "qa", name: "QA Sandbox", is_production: false },
          ]),
          { status: 200 },
        ),
      ),
    );

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <EnvironmentProvider>{children}</EnvironmentProvider>
    );
    const { result } = renderHook(() => useEnvironment(), { wrapper });

    await waitFor(() => expect(result.current.environments).toHaveLength(2));
    expect(result.current.environments[1].slug).toBe("qa");
  });

  it("survives restricted localStorage without throwing", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new DOMException("blocked", "SecurityError");
    });
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new DOMException("blocked", "SecurityError");
    });

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <EnvironmentProvider>{children}</EnvironmentProvider>
    );
    const { result } = renderHook(() => useEnvironment(), { wrapper });

    expect(() => {
      act(() => {
        result.current.setEnvironment("dev");
      });
    }).not.toThrow();
    expect(result.current.currentEnv.id).toBe("dev");
  });
});
