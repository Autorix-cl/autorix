/* eslint-disable */
import * as React from "react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { ThemisPoliciesTable } from "./policies-table";
import { DryRunPlayground } from "./dry-run-playground";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { I18nProvider } from "@/lib/i18n";

vi.mock("@/lib/auth/use-permission", () => ({
  usePermission: () => ({
    operator: { id: "op-1", email: "admin@example.com", name: "Admin", role: "owner", is_local: true, is_active: true },
    isLoading: false,
    can: () => true,
    hasRole: () => true,
    checkSession: vi.fn().mockResolvedValue(true),
  }),
  Can: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

let queryClient = new QueryClient();

// Mock fetch globally
global.fetch = vi.fn();

describe("Themis UI Components", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  });
  it("ThemisPoliciesTable should render policies and allow toggling", async () => {
    const mockPolicies = [
      { id: "1", name: "Admin Rule", expression: "request.role == 'admin'", priority: 1, enabled: true }
    ];
    
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => mockPolicies
    });
    
    render(
      <I18nProvider>
        <QueryClientProvider client={queryClient}>
          <ThemisPoliciesTable />
        </QueryClientProvider>
      </I18nProvider>
    );
    
    // Check if table renders the policy name
    await waitFor(() => {
      expect(screen.getByText("Admin Rule")).toBeInTheDocument();
    });
    
    // Click the toggle switch
    const toggle = screen.getByRole("switch");
    
    // Mock the toggle API response and subsequent query refetch
    (global.fetch as any)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({})
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [{ ...mockPolicies[0], enabled: false }]
      });
    
    fireEvent.click(toggle);
    
    // Verify that fetch was called with the toggle endpoint
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith("/api/themis/policies/1/toggle", expect.objectContaining({
        method: "PATCH"
      }));
    });
  });

  it("DryRunPlayground should display Passed when evaluation is successful", async () => {
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        result: "Passed",
        matchedPolicy: { name: "Test Policy" }
      })
    });
    
    render(
      <QueryClientProvider client={queryClient}>
        <DryRunPlayground />
      </QueryClientProvider>
    );
    
    const runBtn = screen.getByRole("button", { name: /Run Evaluation/i });
    fireEvent.click(runBtn);
    
    await waitFor(() => {
      expect(screen.getByText("Passed")).toBeInTheDocument();
      expect(screen.getByText("Test Policy")).toBeInTheDocument();
    });
  });
});
