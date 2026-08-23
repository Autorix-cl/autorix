import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { IdentityBuilderSheet } from "./identity-builder-sheet";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
);

describe("IdentityBuilderSheet", () => {
  it("renders invite flow without password field", async () => {
    render(<IdentityBuilderSheet isOpen={true} onOpenChange={vi.fn()} />, { wrapper });
    
    // Check header
    expect(screen.getByRole("heading", { name: "Send Invitation" })).toBeInTheDocument();
    
    // Check form fields
    expect(screen.getByLabelText(/Email address/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/First name/i)).toBeInTheDocument();
    
    // Ensure password is gone
    expect(screen.queryByLabelText(/Password/i)).not.toBeInTheDocument();
    expect(screen.getByText(/We no longer allow setting passwords manually/i)).toBeInTheDocument();
  });
});
