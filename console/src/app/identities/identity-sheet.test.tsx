import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect } from "vitest";
import { IdentitySheet } from "./identity-sheet";
import { IdentityItem } from "./columns";

const mockIdentity: IdentityItem = {
  id: "123",
  email: "test@example.com",
  name: "Test User",
  state: "active",
  createdAt: "Recently",
  original: {
    id: "123",
    schema_id: "default",
    traits: { email: "test@example.com" },
    state: "active",
    created_at: "2023-01-01",
    updated_at: "2023-01-01",
  }
};

describe("IdentitySheet", () => {
  it("renders tabs and complete UI", async () => {
    const user = userEvent.setup();
    render(<IdentitySheet identity={mockIdentity} isOpen={true} onOpenChange={() => {}} />);
    
    // Check Tabs
    const securityTab = screen.getByRole("tab", { name: /security/i });
    expect(screen.getByRole("tab", { name: /overview/i })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /traits/i })).toBeInTheDocument();
    expect(securityTab).toBeInTheDocument();
    
    // Check Overview content
    expect(screen.getByText("test@example.com")).toBeInTheDocument();
    
    // Click Security Tab
    await user.click(securityTab);
    
    // Verify Security Actions exist
    await waitFor(() => {
      expect(screen.getByText("Force Password Reset")).toBeInTheDocument();
      expect(screen.getByText("Active Sessions")).toBeInTheDocument();
      expect(screen.getByText("Enrolled Factors")).toBeInTheDocument();
      expect(screen.getByText("Suspend Account")).toBeInTheDocument();
    });
  });
});
