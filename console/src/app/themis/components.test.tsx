import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { ThemisPoliciesTable } from "./policies-table";
import { DryRunPlayground } from "./dry-run-playground";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const queryClient = new QueryClient();

// Mock fetch globally
global.fetch = vi.fn();

describe("Themis UI Components", () => {
  it("ThemisPoliciesTable should render policies and allow toggling", async () => {
    const mockPolicies = [
      { id: "1", name: "Admin Rule", expression: "request.role == 'admin'", priority: 1, enabled: true }
    ];
    
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => mockPolicies
    });
    
    render(
      <QueryClientProvider client={queryClient}>
        <ThemisPoliciesTable />
      </QueryClientProvider>
    );
    
    // Check if table renders the policy name
    await waitFor(() => {
      expect(screen.getByText("Admin Rule")).toBeInTheDocument();
    });
    
    // Click the toggle switch
    const toggle = screen.getByRole("switch");
    
    // Mock the toggle API response
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => ({})
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
