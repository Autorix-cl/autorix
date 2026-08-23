import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { KeyBuilderSheet } from "./key-builder-sheet";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

// Mock matchMedia for Radix UI
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(), // deprecated
    removeListener: vi.fn(), // deprecated
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
window.ResizeObserver = ResizeObserver;

const queryClient = new QueryClient();

describe("KeyBuilderSheet", () => {
  it("should render create button", () => {
    render(
      <QueryClientProvider client={queryClient}>
        <KeyBuilderSheet />
      </QueryClientProvider>
    );
    expect(screen.getByRole("button", { name: /Create API Key/i })).toBeInTheDocument();
  });
});
