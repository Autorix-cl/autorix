import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { createQueryClient, AppQueryProvider } from "./provider";

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
