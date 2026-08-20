import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import * as React from "react";
import { EnvironmentProvider, useEnvironment } from "./environment-context";

describe("EnvironmentContext", () => {
  beforeEach(() => {
    localStorage.clear();
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
});
