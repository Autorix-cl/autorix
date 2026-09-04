import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { createQueryClient, AppQueryProvider } from "./provider";
import { setRedirectHandler, resetRedirectFlag } from "../api/client";

describe("createQueryClient", () => {
  it("sets a non-zero stale time so identical queries are deduplicated instead of refetched on every mount", () => {
    const client = createQueryClient();
    const defaults = client.getDefaultOptions();
    expect(defaults.queries?.staleTime).toBeGreaterThan(0);
  });

  it("enables a bounded retry policy instead of retrying forever or not at all", () => {
    const client = createQueryClient();
    const defaults = client.getDefaultOptions();
    expect(defaults.queries?.retry).toBe(2);
  });

  it("refetches on window focus so a stale dashboard catches up when the tab regains focus", () => {
    const client = createQueryClient();
    const defaults = client.getDefaultOptions();
    expect(defaults.queries?.refetchOnWindowFocus).toBe(true);
  });

  it("triggers redirectToLogin when queryCache experiences an unauthorized error", () => {
    const redirectSpy = vi.fn();
    setRedirectHandler(redirectSpy);
    const client = createQueryClient();

    // Trigger onError directly on queryCache
    const queryCache = client.getQueryCache();
    // @ts-expect-error accessing internal config
    queryCache.config.onError?.({ kind: "unauthorized", status: 401, message: "unauthorized" });

    expect(redirectSpy).toHaveBeenCalledTimes(1);
    expect(redirectSpy).toHaveBeenCalledWith(expect.stringContaining("/login?from="));
    resetRedirectFlag();
  });
});

describe("AppQueryProvider", () => {
  it("renders its children", () => {
    render(
      <AppQueryProvider>
        <div>hello from inside the provider</div>
      </AppQueryProvider>,
    );
    expect(screen.getByText("hello from inside the provider")).toBeInTheDocument();
  });
});
